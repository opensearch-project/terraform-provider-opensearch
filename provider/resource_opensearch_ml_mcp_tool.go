package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// The ML Commons MCP server exposes four endpoints. Register, update and remove landed in
// OpenSearch 3.0; _list landed in 3.1. Terraform cannot manage what it cannot read, so the
// effective floor for this resource is 3.1.
const (
	mcpToolsRegisterPath = "/_plugins/_ml/mcp/tools/_register"
	mcpToolsUpdatePath   = "/_plugins/_ml/mcp/tools/_update"
	mcpToolsListPath     = "/_plugins/_ml/mcp/tools/_list"
	mcpToolsRemovePath   = "/_plugins/_ml/mcp/tools/_remove"
)

// Message fragments returned by ML Commons. These are matched on text rather than on status
// code because the MCP transport actions do not follow the provider's usual status
// conventions — see the comments at each use site. All three were captured against
// opensearchproject/opensearch:3 (3.8.0).
const (
	// Returned by every MCP endpoint while plugins.ml_commons.mcp_server_enabled is false.
	// The endpoint answers HTTP 500, so the fragment and not the status identifies it.
	mcpServerDisabledFragment = "The MCP server is not enabled"

	// Register rejects a name that is already taken. Both the wording and the status vary by
	// release, so neither can be relied on alone:
	//   3.1.0-3.4.0  HTTP 500 "Unable to register tools: [X] as they already exist"
	//   3.5.0-3.8.0  HTTP 400 "Unable to register tool: a tool with the same name already exists."
	// "already exist" is the fragment common to both.
	mcpToolDuplicateNameFragment = "already exist"

	// Registering or updating writes to the system index first and only then loads the tool
	// into each node's MCP server memory. When the second step fails — for example a PPLTool
	// with no model_id, which 3.1.0-3.7.0 reject at load time — the API returns an error even
	// though the tool is already persisted and visible in _list. Create must claim the ID on
	// this path or the tool is orphaned: it exists in the registry, Terraform does not know
	// about it, and the next apply fails as a duplicate.
	//   3.1.0-3.4.0  "Tools: [X] are persisted successfully but failed to register to mcp server memory with error: ..."
	//   3.5.0-3.7.0  "Tools are persisted successfully but failed to register to mcp server memory"
	mcpToolPersistedNotLoadedFragment = "but failed to register to mcp server memory"

	// The update-side equivalent. No orphan risk, since the ID already exists, but the tool in
	// the registry no longer matches what the MCP server is serving.
	//   3.1.0-3.4.0  "Tools: [X] are updated successfully but failed to update to mcp server memory with error: ..."
	//   3.5.0-3.7.0  "Tools are updated successfully, but failed to update to mcp server memory"
	mcpToolUpdatedNotLoadedFragment = "but failed to update to mcp server memory"

	// Remove is the mirror of the register/update half-success: it deletes from the system index
	// first and only then drops the tool from each node's MCP server memory. When the tool was
	// never loaded into memory — the state a failed register leaves behind — the second step
	// reports "not found on node" and the whole call errors, even though the registry entry, which
	// is what this resource manages, is already gone. Verified on 3.5.0: after this error `_list`
	// no longer returns the tool.
	//
	// Both fragments are required. "removed successfully in index" alone would also swallow a
	// memory-removal failure with a different cause — an unreachable node, say — where the tool
	// could still be live in some node's memory and reporting a clean destroy would be a lie.
	// Pairing it with "not found on node" narrows this to the benign case: nothing was in memory
	// to remove. Any other cause still surfaces, and because the index row is already gone a
	// second destroy then succeeds through mcpToolNotInSystemIndexFragment.
	mcpToolRemovedFromIndexFragment = "removed successfully in index"
	mcpToolNotInNodeMemoryFragment  = "not found on node"

	// _list fails outright when the system index has never been created, on every release
	// before 3.8.0:
	//   3.1.0-3.7.0  HTTP 500 "Failed to search mcp tools index with error: no such index [.plugins-ml-mcp-tools]"
	//   3.8.0        HTTP 200 {"tools":[]}
	// An absent index means an empty registry, so this is mapped onto errMCPToolNotFound
	// rather than surfaced. Without it, read errors instead of reporting the tool absent.
	mcpToolsIndexAbsentFragment = "no such index"

	// Returned by OpenSearch when a URI has no registered handler. Every MCP endpoint answers
	// this on a pre-3.0 cluster, and `_list` alone answers it on 3.0, which is why the resource
	// floor is 3.1 rather than 3.0.
	mcpEndpointAbsentFragment = "no handler found for uri"

	// TransportMcpToolsRemoveAction raises OpenSearchException when none of the requested
	// names is present, which surfaces as HTTP 500 and *not* as HTTP 404. The wording has
	// already changed once upstream ("these tools: %s are not found in system index" in
	// earlier releases, "no tool in the request found in system index" on 3.8), so the
	// matcher deliberately keys on the fragment common to both.
	mcpToolNotInSystemIndexFragment = "found in system index"
)

// mcpToolRequestTimeout bounds the retry of a transient failure. Shard initialization takes
// well under a second on an idle cluster, so this only needs to cover a loaded CI runner; it is
// deliberately far shorter than the SDK's 20-minute default so a genuinely broken cluster fails
// the apply promptly instead of hanging it.
const mcpToolRequestTimeout = 2 * time.Minute

// errMCPToolNotFound is the sentinel getMCPToolFromAPI returns when _list succeeds but holds
// no entry for the requested name. Callers test it with errors.Is; there is no get-by-name
// endpoint whose 404 could be used instead.
var errMCPToolNotFound = errors.New("MCP tool not registered")

func resourceOpensearchMLMCPTool() *schema.Resource {
	return &schema.Resource{
		Description:   "OpenSearch ML Commons MCP tool resource. Manages a single tool in the registry of the built-in MCP (Model Context Protocol) server, which determines the tools an MCP client can discover and invoke. Requires OpenSearch 3.1 or later — the register, update and remove APIs arrived in 3.0, but the List API this resource reads through arrived in 3.1 — and requires the cluster setting `plugins.ml_commons.mcp_server_enabled` to be `true`. That setting is not exposed by `opensearch_cluster_settings`, which carries a fixed list of settings, so enable it in `opensearch.yml` or with `PUT /_cluster/settings` before using this resource.\n\nA note on the update API: it merges rather than replaces, and neither an empty value nor an explicit null clears a field. `description` can still be cleared, because the empty string is an accepted value, but removing `parameters` or `attributes` from the configuration forces the tool to be replaced.",
		CreateContext: resourceOpensearchMLMCPToolCreate,
		ReadContext:   resourceOpensearchMLMCPToolRead,
		UpdateContext: resourceOpensearchMLMCPToolUpdate,
		DeleteContext: resourceOpensearchMLMCPToolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceOpensearchMLMCPToolCustomizeDiff,
		Schema: map[string]*schema.Schema{
			// ============================================
			// ===         Required attributes          ===
			// ============================================
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the tool, unique within the registry. This is the registry key — register rejects duplicates, update matches on it, and remove takes a list of it — so it is also the resource ID. Changing it registers a different tool.",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Tool type, for example `\"ListIndexTool\"`, `\"IndexMappingTool\"`, or `\"SearchIndexTool\"`. Selects the tool implementation, and is therefore not changed in place.",
			},

			// ============================================
			// ===         Optional attributes          ===
			// ============================================
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the tool, surfaced to MCP clients during discovery. The API does not supply a default, so leaving this unset leaves the tool without a description.",
			},
			"parameters": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Tool parameters, as a flat map of strings — for example `model_id` and `model_type` for `\"PPLTool\"`. Nested parameters are not supported. Removing this attribute forces replacement, because the update API cannot clear it.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"attributes": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Tool attributes as a JSON document, holding the `input_schema` JSON Schema that describes the tool's arguments. Write it with `jsonencode()`. Removing this attribute forces replacement, because the update API cannot clear it.",
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: diffSuppressMCPToolAttributes,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
		},
	}
}

// The update API merges: a field omitted from the request keeps its stored value, and neither
// an empty object nor an explicit null clears it (verified on 3.8.0). `description` can still
// be cleared, because the empty string is itself an accepted value, but `parameters` and
// `attributes` cannot. Removing either from the configuration would otherwise produce a plan
// that never converges — read keeps bringing the stored value back — so removal forces
// replacement instead.
func resourceOpensearchMLMCPToolCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	if d.Id() == "" {
		return nil
	}

	oldParameters, newParameters := d.GetChange("parameters")
	if len(oldParameters.(map[string]interface{})) > 0 && len(newParameters.(map[string]interface{})) == 0 {
		if err := d.ForceNew("parameters"); err != nil {
			return err
		}
	}

	oldAttributes, newAttributes := d.GetChange("attributes")
	if oldAttributes.(string) != "" && newAttributes.(string) == "" {
		if err := d.ForceNew("attributes"); err != nil {
			return err
		}
	}

	return nil
}

func resourceOpensearchMLMCPToolCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	name := d.Get("name").(string)

	jsonPayload, err := json.Marshal(map[string]interface{}{
		"tools": []interface{}{buildMCPToolPayload(d)},
	})
	if err != nil {
		return diag.Errorf("failed to marshal MCP tool payload: %s", err)
	}

	result, err := performMCPToolRequest(ctx, conf, "POST", mcpToolsRegisterPath, string(jsonPayload), "register MCP tool")
	if err != nil {
		// The tool is already in the registry on this path, so claim the ID before reporting the
		// failure. Terraform records it as tainted: the next apply replaces it, and destroy can
		// clean it up. Returning without an ID would leave it orphaned.
		if strings.Contains(err.Error(), mcpToolPersistedNotLoadedFragment) {
			d.SetId(name)
			return diag.Errorf("%s\n\nThe tool was written to the registry but the MCP server could not load it, so it is not yet usable by MCP clients. It has been recorded in state as tainted; correct the configuration — a `PPLTool` needs a `model_id` in `parameters`, for instance — and apply again to replace it.", err)
		}
		if strings.Contains(err.Error(), mcpToolDuplicateNameFragment) {
			return diag.Errorf("%s\n\nA tool named %q is already registered. Adopt it into state instead of registering it again:\n\n    terraform import opensearch_ml_mcp_tool.<label> %s", err, name, name)
		}
		return diagnoseMCPToolError(err)
	}

	// The tool is written to the .plugins-ml-mcp-tools system index before the nodes sync it,
	// so a partial failure has already left an object behind. Claim the ID before reporting
	// the failure, so the next apply or a destroy can clean up rather than orphaning it.
	d.SetId(name)

	if err := validateMCPToolNodeResponse(result, "created", "register MCP tool"); err != nil {
		return diag.FromErr(err)
	}

	return resourceOpensearchMLMCPToolRead(ctx, d, m)
}

func resourceOpensearchMLMCPToolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	tool, err := getMCPToolFromAPI(ctx, conf, d.Id())
	if err != nil {
		if errors.Is(err, errMCPToolNotFound) {
			d.SetId("")
			return nil
		}
		return diagnoseMCPToolError(err)
	}

	// ============================================
	// === Required attributes
	// ============================================
	if name, ok := tool["name"].(string); ok {
		if err := d.Set("name", name); err != nil {
			return diag.Errorf("error setting name: %s", err)
		}
	}
	if toolType, ok := tool["type"].(string); ok {
		if err := d.Set("type", toolType); err != nil {
			return diag.Errorf("error setting type: %s", err)
		}
	}

	// ============================================
	// === Optional attributes
	// ============================================
	// The API backfills none of these, so an attribute missing from the response genuinely is
	// unset on the cluster and is cleared in state rather than left stale.
	description, _ := tool["description"].(string)
	if err := d.Set("description", description); err != nil {
		return diag.Errorf("error setting description: %s", err)
	}

	parameters, _ := tool["parameters"].(map[string]interface{})
	if err := d.Set("parameters", parameters); err != nil {
		return diag.Errorf("error setting parameters: %s", err)
	}

	attributes := ""
	if raw, ok := tool["attributes"]; ok && raw != nil {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return diag.Errorf("error serializing attributes: %s", err)
		}
		// The API does not preserve key ordering, so normalize before comparing with the
		// configured value, which is stored normalized by the schema's StateFunc.
		attributes, err = structure.NormalizeJsonString(string(encoded))
		if err != nil {
			return diag.Errorf("error normalizing attributes: %s", err)
		}
	}
	if err := d.Set("attributes", attributes); err != nil {
		return diag.Errorf("error setting attributes: %s", err)
	}

	return nil
}

func resourceOpensearchMLMCPToolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload := buildMCPToolPayload(d)
	// buildMCPToolPayload omits an empty description, which is right for register but would
	// make clearing one impossible on update: the API merges, so an omitted field keeps its
	// stored value. The empty string is an accepted value, so send it explicitly.
	payload["description"] = d.Get("description").(string)

	jsonPayload, err := json.Marshal(map[string]interface{}{
		"tools": []interface{}{payload},
	})
	if err != nil {
		return diag.Errorf("failed to marshal MCP tool update payload: %s", err)
	}

	result, err := performMCPToolRequest(ctx, conf, "POST", mcpToolsUpdatePath, string(jsonPayload), "update MCP tool")
	if err != nil {
		if strings.Contains(err.Error(), mcpToolUpdatedNotLoadedFragment) {
			return diag.Errorf("%s\n\nThe registry was updated but the MCP server could not load the new definition, so MCP clients are still being served the previous one. Correct the configuration and apply again.", err)
		}
		return diagnoseMCPToolError(err)
	}

	if err := validateMCPToolNodeResponse(result, "updated", "update MCP tool"); err != nil {
		return diag.FromErr(err)
	}

	return resourceOpensearchMLMCPToolRead(ctx, d, m)
}

func resourceOpensearchMLMCPToolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	// The remove API takes a bare JSON array of names, not an object with a "tools" key.
	jsonPayload, err := json.Marshal([]string{d.Id()})
	if err != nil {
		return diag.Errorf("failed to marshal MCP tool removal payload: %s", err)
	}

	result, err := performMCPToolRequest(ctx, conf, "POST", mcpToolsRemovePath, string(jsonPayload), "remove MCP tool")
	if err != nil {
		// A tool that is already gone is not an error for destroy. Note that this cannot use
		// the errors.As/StatusNotFound idiom the other ML resources use: ML Commons reports an
		// unknown tool as an OpenSearchException, which surfaces as HTTP 500. See
		// mcpToolNotInSystemIndexFragment for the upstream source and the wording history.
		if strings.Contains(err.Error(), mcpToolNotInSystemIndexFragment) {
			return nil
		}
		// The registry entry is gone and nothing was in memory to remove, which is all destroy
		// needs. A memory-removal failure from any other cause deliberately falls through.
		if strings.Contains(err.Error(), mcpToolRemovedFromIndexFragment) &&
			strings.Contains(err.Error(), mcpToolNotInNodeMemoryFragment) {
			return nil
		}
		return diagnoseMCPToolError(err)
	}

	if err := validateMCPToolNodeResponse(result, "removed", "remove MCP tool"); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// ============================================
// ===         Helper functions             ===
// ============================================

func buildMCPToolPayload(d *schema.ResourceData) map[string]interface{} {
	payload := map[string]interface{}{
		"name": d.Get("name").(string),
		"type": d.Get("type").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		payload["description"] = v.(string)
	}
	if v, ok := d.GetOk("parameters"); ok {
		payload["parameters"] = v.(map[string]interface{})
	}
	if v, ok := d.GetOk("attributes"); ok {
		// Validated as JSON by the schema, so a parse failure here is not reachable through a
		// plan; ignoring it keeps attributes out of the payload rather than sending a string
		// where the API expects an object.
		var attributes interface{}
		if err := json.Unmarshal([]byte(v.(string)), &attributes); err == nil {
			payload["attributes"] = attributes
		}
	}

	return payload
}

// isTransientMCPToolError reports whether a failure is expected to clear on its own.
//
// Register, update and remove all search .plugins-ml-mcp-tools to resolve tool names, and that
// search fails while the index is still being created. A single apply that registers several
// tools — the allowlist this resource exists for — races on exactly that: the first request
// creates the index and the rest fail with "all shards failed". Reproduced on every release
// from 3.1.0 through 3.8.0.
//
// Deliberately narrow. An absent index is *not* included: for _list that means an empty
// registry, which getMCPToolFromAPI reports as errMCPToolNotFound rather than retrying until
// the timeout expires.
func isTransientMCPToolError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "all shards failed") ||
		strings.Contains(message, "no shard available")
}

// performMCPToolRequest issues an MCP request, retrying while the failure looks transient. The
// body is passed as a string rather than an io.Reader because a Reader cannot be replayed on a
// second attempt.
func performMCPToolRequest(ctx context.Context, conf *ProviderConf, method, path, body, operation string) (map[string]interface{}, error) {
	url := conf.rawUrl + path

	var result map[string]interface{}
	err := retry.RetryContext(ctx, mcpToolRequestTimeout, func() *retry.RetryError {
		var payload io.Reader
		if body != "" {
			payload = strings.NewReader(body)
		}

		res, err := performRequestAndParse(ctx, conf.osClient, method, url, payload, operation)
		if err != nil {
			if isTransientMCPToolError(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// getMCPToolFromAPI returns the registry entry for name. There is no get-by-name endpoint, so
// this lists the whole registry and filters client-side; the registry is an operator-curated
// allowlist, so the scan is cheap. Returns errMCPToolNotFound when the name is absent, which
// both read and the acceptance tests' CheckDestroy helper test for.
func getMCPToolFromAPI(ctx context.Context, conf *ProviderConf, name string) (map[string]interface{}, error) {
	result, err := performMCPToolRequest(ctx, conf, "GET", mcpToolsListPath, "", "list MCP tools")
	if err != nil {
		// Releases before 3.8.0 fail rather than answering an empty list when the system index
		// has never been created. An absent index is an empty registry, not an error.
		if strings.Contains(err.Error(), mcpToolsIndexAbsentFragment) {
			return nil, fmt.Errorf("%w: %s", errMCPToolNotFound, name)
		}
		return nil, err
	}

	// From 3.8.0 McpToolsHelper.searchAllTools catches IndexNotFoundException and answers
	// {"tools":[]}, so a cluster where nothing was ever registered lands here instead.
	tools, ok := result["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("tools array not found in list MCP tools response: %v", result)
	}

	for _, raw := range tools {
		tool, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if toolName, _ := tool["name"].(string); toolName == name {
			return tool, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errMCPToolNotFound, name)
}

// validateMCPToolNodeResponse checks a per-node MCP response of the form
// {"<nodeId>":{"created":true},...}. These APIs carry no top-level status, and a cluster can
// legitimately answer success for one node and failure for another, so success is not inferred
// from the HTTP status or from the first entry.
func validateMCPToolNodeResponse(result map[string]interface{}, key, operation string) error {
	if len(result) == 0 {
		return fmt.Errorf("%s returned an empty response; expected at least one node to report %q", operation, key)
	}

	var failed []string
	for nodeID, raw := range result {
		status, ok := raw.(map[string]interface{})
		if !ok {
			failed = append(failed, nodeID)
			continue
		}
		if reported, _ := status[key].(bool); !reported {
			failed = append(failed, nodeID)
		}
	}

	if len(failed) > 0 {
		sort.Strings(failed)
		return fmt.Errorf("%s did not report %q on %d of %d node(s): %s", operation, key, len(failed), len(result), strings.Join(failed, ", "))
	}

	return nil
}

// diagnoseMCPToolError wraps failures an operator can act on with the corrective action.
func diagnoseMCPToolError(err error) diag.Diagnostics {
	// Matched on the message rather than the status code, which is 400 here — the same status a
	// working cluster uses for ordinary bad requests, so branching on it would mislabel them.
	if strings.Contains(err.Error(), mcpEndpointAbsentFragment) {
		return diag.Errorf("%s\n\nThis cluster has no MCP tools API. `opensearch_ml_mcp_tool` requires OpenSearch 3.1 or later: the register, update and remove APIs arrived in 3.0, but reading state depends on `GET %s`, which arrived in 3.1.", err, mcpToolsListPath)
	}
	if strings.Contains(err.Error(), mcpServerDisabledFragment) {
		return diag.Errorf("%s\n\nThe ML Commons MCP server is disabled on this cluster. Set `plugins.ml_commons.mcp_server_enabled` to `true` in `opensearch.yml`, or dynamically with `PUT /_cluster/settings {\"persistent\":{\"plugins.ml_commons.mcp_server_enabled\":\"true\"}}`. The `opensearch_cluster_settings` resource does not expose this setting.", err)
	}
	return diag.FromErr(err)
}
