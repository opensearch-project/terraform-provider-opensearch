package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Maps of resource IDs and registry create times, used to tell an in-place update apart from a
// replacement. Note: these global maps are shared across tests.
var savedMLMCPToolIDs = make(map[string]string)
var savedMLMCPToolCreateTimes = make(map[string]float64)

// mcpToolsAvailability caches a single probe of the cluster. The MCP tools List API arrived in
// OpenSearch 3.1, and every MCP endpoint short-circuits while
// plugins.ml_commons.mcp_server_enabled is false. Probing the endpoint covers both cases,
// where comparing conf.osVersion would only cover the first — and version detection is
// best-effort anyway, since opensearch_version can be set by hand and healthcheck can be off.
var mcpToolsAvailability struct {
	once   sync.Once
	reason string
}

func skipIfMCPToolsUnavailable(t *testing.T) {
	mcpToolsAvailability.once.Do(func() {
		provider := Provider()
		if diags := provider.Configure(context.Background(), &terraform.ResourceConfig{}); diags.HasError() {
			mcpToolsAvailability.reason = fmt.Sprintf("%#v", diags)
			return
		}
		conf, ok := provider.Meta().(*ProviderConf)
		if !ok {
			mcpToolsAvailability.reason = "provider did not yield a ProviderConf"
			return
		}
		// A successful list that simply has no such tool is the healthy answer.
		if _, err := getMCPToolFromAPI(context.Background(), conf, "terraform-provider-availability-probe"); err != nil && !errors.Is(err, errMCPToolNotFound) {
			mcpToolsAvailability.reason = err.Error()
		}
	})

	if mcpToolsAvailability.reason != "" {
		t.Skipf("MCP tools API only supported on OS >= 3.1 with plugins.ml_commons.mcp_server_enabled=true: %s", mcpToolsAvailability.reason)
	}
}

// TestOpensearchMLMCPToolForceNewAttributes pins the immutable attributes at the schema level.
// It needs no cluster, so it also runs on the 2.x matrix entry.
func TestOpensearchMLMCPToolForceNewAttributes(t *testing.T) {
	toolSchema := resourceOpensearchMLMCPTool().Schema
	for _, attribute := range []string{"name", "type"} {
		if !toolSchema[attribute].ForceNew {
			t.Errorf("%q must be ForceNew: it selects the tool's identity or implementation, which the update API does not change in place", attribute)
		}
	}
	for _, attribute := range []string{"description", "parameters", "attributes"} {
		if toolSchema[attribute].ForceNew {
			t.Errorf("%q must not be ForceNew: it is what the update API exists to change", attribute)
		}
	}
}

func TestAccOpensearchMLMCPTool_Minimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_Minimal(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.minimal", "id", "tf_acc_minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.minimal", "name", "tf_acc_minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.minimal", "type", "ListIndexTool"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.minimal", "description", ""),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.minimal", "attributes", ""),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.minimal", "parameters.%", "0"),
				),
			},
			{
				ResourceName:      "opensearch_ml_mcp_tool.minimal",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLMCPTool_WithOptionalAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_WithOptionalAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.with_optional", "name", "tf_acc_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.with_optional", "type", "ListIndexTool"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.with_optional", "description", "Lists indices"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.with_optional", "parameters.model_type", "FINETUNE"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.with_optional", "parameters.execute", "false"),
					testCheckOpensearchMLMCPToolAttributesEquivalent(
						"opensearch_ml_mcp_tool.with_optional",
						`{"input_schema":{"type":"object","properties":{"question":{"type":"string","description":"The question in natural language"}},"required":["question"]}}`,
					),
				),
			},
			{
				ResourceName:      "opensearch_ml_mcp_tool.with_optional",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLMCPTool_Update(t *testing.T) {
	defer func() {
		delete(savedMLMCPToolIDs, "opensearch_ml_mcp_tool.update")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_Update(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.update"),
					testSaveOpensearchMLMCPToolID("opensearch_ml_mcp_tool.update"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.update", "description", "Lists indices"),
					testCheckOpensearchMLMCPToolAttributesEquivalent(
						"opensearch_ml_mcp_tool.update",
						`{"input_schema":{"type":"object","properties":{"indices":{"type":"array","items":{"type":"string"}}}}}`,
					),
				),
			},
			{
				Config: testAccOpensearchMLMCPToolConfig_Update_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.update"),
					testCheckOpensearchMLMCPToolIDEqualsSavedID("opensearch_ml_mcp_tool.update"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.update", "description", "Lists indices, with a local filter"),
					testCheckOpensearchMLMCPToolAttributesEquivalent(
						"opensearch_ml_mcp_tool.update",
						`{"input_schema":{"type":"object","properties":{"indices":{"type":"array","items":{"type":"string"}},"local":{"type":"boolean"}}}}`,
					),
				),
			},
		},
	})
}

// A replacement is invisible in state when `type` changes, because the resource ID is the tool
// name and the name is unchanged. The registry's create_time is the observable difference: a
// re-registration stamps a new one, where an in-place update would keep it and add
// last_update_time instead.
func TestAccOpensearchMLMCPTool_ForceNew(t *testing.T) {
	defer func() {
		delete(savedMLMCPToolCreateTimes, "opensearch_ml_mcp_tool.force_new")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_ForceNew("ListIndexTool"),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.force_new"),
					testSaveOpensearchMLMCPToolCreateTime("opensearch_ml_mcp_tool.force_new"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.force_new", "type", "ListIndexTool"),
				),
			},
			{
				Config: testAccOpensearchMLMCPToolConfig_ForceNew("IndexMappingTool"),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.force_new"),
					resource.TestCheckResourceAttr("opensearch_ml_mcp_tool.force_new", "type", "IndexMappingTool"),
					testCheckOpensearchMLMCPToolCreateTimeChanged("opensearch_ml_mcp_tool.force_new"),
				),
			},
		},
	})
}

// TestAccOpensearchMLMCPTool_ParallelCreate pins the transient-failure retry. Terraform creates
// independent resources concurrently, so a first apply that registers an allowlist of tools has
// several registers in flight at once. The first one creates the .plugins-ml-mcp-tools index and
// the rest race against it: ml-commons searches that index to resolve tool names, and the search
// fails with "all shards failed" while it is still initializing. Without the retry in
// performMCPToolRequest every tool but one fails, on every release from 3.1.0 to 3.8.0.
func TestAccOpensearchMLMCPTool_ParallelCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_Parallel(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.parallel_a"),
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.parallel_b"),
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.parallel_c"),
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.parallel_d"),
				),
			},
		},
	})
}

func TestAccOpensearchMLMCPTool_AttributesJSONNormalization(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_Normalization_Ordered(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.normalization"),
				),
			},
			{
				// The same document with the keys reordered and the whitespace changed. A
				// non-empty plan here fails the step, which is the assertion.
				Config:   testAccOpensearchMLMCPToolConfig_Normalization_Reordered(),
				PlanOnly: true,
			},
		},
	})
}

// TestAccOpensearchMLMCPTool_InvalidAttributesJSON pins the plan-time rejection of an
// `attributes` value that is not valid JSON. validation.StringIsJSON runs while the plan is
// built, before the provider is asked to do anything, so the step is PlanOnly: the error has to
// come from schema validation and not from the API rejecting the document. That also means the
// test needs no MCP-capable cluster and runs on every matrix entry.
func TestAccOpensearchMLMCPTool_InvalidAttributesJSON(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchMLMCPToolConfig_InvalidAttributes(),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`"attributes" contains an invalid JSON`),
			},
		},
	})
}

// TestAccOpensearchMLMCPTool_DestroyAfterOutOfBandRemoval pins the delete matcher. ML Commons
// reports removing an absent tool as HTTP 500 carrying "found in system index", not as a 404,
// so this fails loudly if the upstream wording changes rather than silently breaking destroy.
func TestAccOpensearchMLMCPTool_DestroyAfterOutOfBandRemoval(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_OutOfBand(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLMCPToolExists("opensearch_ml_mcp_tool.out_of_band"),
					// Remove it behind Terraform's back. The implicit destroy at the end of the
					// test then has nothing to remove and must still succeed.
					testAccRemoveMCPToolOutOfBand("opensearch_ml_mcp_tool.out_of_band"),
				),
				// The resource is gone from the cluster but still in state, so refreshing at the
				// end of the step legitimately plans to recreate it.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccOpensearchMLMCPTool_DestroyAfterFailedLoad pins the delete-side half-success.
//
// Register writes to the system index and then loads the tool into each node's MCP server
// memory. When the load fails the tool is persisted but not in memory, and remove later reports
// "removed successfully in index but failed to remove from mcp server in memory ... not found on
// node". Treating that as a failure makes such a tool impossible to destroy — it stays in state
// forever, because each retry removes nothing and errors identically.
//
// Producing the state needs a release that rejects the load: 3.1.0-3.7.0 refuse a PPLTool with
// no model_id, 3.8.0 accepts it, so the test probes first and skips where it cannot apply.
func TestAccOpensearchMLMCPTool_DestroyAfterFailedLoad(t *testing.T) {
	testAccPreCheck(t)
	skipIfMCPToolsUnavailable(t)
	if !mcpToolLoadFailureReproducible(t) {
		t.Skip("this cluster accepts a PPLTool with no model_id, so the failed-load state cannot be produced")
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfMCPToolsUnavailable(t)
		},
		Providers: testAccOpendistroProviders,
		// The destroy that runs at the end of the test is the assertion: it must succeed even
		// though the tool was never loaded into MCP server memory.
		CheckDestroy: testCheckOpensearchMLMCPToolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLMCPToolConfig_FailedLoad(),
				// Create claims the ID before returning this error, so the tool stays under
				// management rather than being orphaned in the registry.
				ExpectError: regexp.MustCompile("recorded in state as tainted"),
			},
		},
	})
}

// mcpToolLoadFailureReproducible reports whether this cluster rejects a PPLTool that has no
// model_id when loading it into MCP server memory. It registers one directly, inspects the
// outcome, and cleans up either way.
func mcpToolLoadFailureReproducible(t *testing.T) bool {
	conf := testAccOpendistroProvider.Meta().(*ProviderConf)
	ctx := context.Background()
	const probe = "tf_acc_load_failure_probe"

	body, err := json.Marshal(map[string]interface{}{
		"tools": []interface{}{map[string]interface{}{"type": "PPLTool", "name": probe}},
	})
	if err != nil {
		t.Fatalf("failed to build probe payload: %s", err)
	}

	_, registerErr := performMCPToolRequest(ctx, conf, "POST", mcpToolsRegisterPath, string(body), "probe register MCP tool")

	// Registered or not, the tool may now be in the index; remove it so the probe leaves nothing.
	if removal, err := json.Marshal([]string{probe}); err == nil {
		_, _ = performMCPToolRequest(ctx, conf, "POST", mcpToolsRemovePath, string(removal), "probe remove MCP tool")
	}

	return registerErr != nil && strings.Contains(registerErr.Error(), mcpToolPersistedNotLoadedFragment)
}

// ============================================
// ===         Check functions              ===
// ============================================

func testSaveOpensearchMLMCPToolID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		savedMLMCPToolIDs[name] = resourceID
		return nil
	}
}

func testCheckOpensearchMLMCPToolIDEqualsSavedID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		currentID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}

		savedID, ok := savedMLMCPToolIDs[name]
		if !ok {
			return fmt.Errorf("resource with name '%s' not found in savedMLMCPToolIDs", name)
		}

		if savedID != currentID {
			return fmt.Errorf("ID of MCP tool with name %s does not match original resource. ID of original resource: %s. ID of resource after update: %s", name, savedID, currentID)
		}

		return nil
	}
}

func testCheckOpensearchMLMCPToolExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMCPToolFromAPI(context.Background(), conf, resourceID)
		return apiError
	}
}

// testCheckOpensearchMLMCPToolAttributesEquivalent compares the `attributes` attribute in state
// with an expected document structurally: the List API does not preserve key ordering, so a
// textual TestCheckResourceAttr would be brittle.
func testCheckOpensearchMLMCPToolAttributesEquivalent(name, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource '%s' not found in state", name)
		}

		var got, want interface{}
		if err := json.Unmarshal([]byte(rs.Primary.Attributes["attributes"]), &got); err != nil {
			return fmt.Errorf("attributes in state is not valid JSON: %s", err)
		}
		if err := json.Unmarshal([]byte(expected), &want); err != nil {
			return fmt.Errorf("expected attributes is not valid JSON: %s", err)
		}
		if !jsonEquivalent(got, want) {
			return fmt.Errorf("attributes of '%s' do not match. got: %s, want: %s", name, rs.Primary.Attributes["attributes"], expected)
		}
		return nil
	}
}

func testSaveOpensearchMLMCPToolCreateTime(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		createTime, err := getMCPToolCreateTime(s, name)
		if err != nil {
			return err
		}
		savedMLMCPToolCreateTimes[name] = createTime
		return nil
	}
}

func testCheckOpensearchMLMCPToolCreateTimeChanged(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		createTime, err := getMCPToolCreateTime(s, name)
		if err != nil {
			return err
		}

		saved, ok := savedMLMCPToolCreateTimes[name]
		if !ok {
			return fmt.Errorf("resource with name '%s' not found in savedMLMCPToolCreateTimes", name)
		}

		if createTime == saved {
			return fmt.Errorf("MCP tool '%s' kept its create_time (%v) across a change to a ForceNew attribute, so it was updated in place instead of replaced", name, saved)
		}

		return nil
	}
}

func testAccRemoveMCPToolOutOfBand(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		body, err := json.Marshal([]string{resourceID})
		if err != nil {
			return err
		}
		if _, err := performRequestAndParse(context.Background(), conf.osClient, "POST", conf.rawUrl+mcpToolsRemovePath, strings.NewReader(string(body)), "remove MCP tool out of band"); err != nil {
			return err
		}
		return nil
	}
}

func testCheckOpensearchMLMCPToolDestroy(s *terraform.State) error {
	resourceType := "opensearch_ml_mcp_tool"
	for _, rs := range s.RootModule().Resources {
		if rs.Type != resourceType {
			continue
		}

		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMCPToolFromAPI(context.Background(), conf, rs.Primary.ID)

		if apiError == nil {
			return fmt.Errorf("resource of type %s with ID '%s' still exists", resourceType, rs.Primary.ID)
		}
		// Deliberately not the strings.Contains(err, "404") idiom the other ML tests use: there
		// is no get-by-name endpoint here, so absence is the errMCPToolNotFound sentinel that
		// getMCPToolFromAPI returns when _list holds no matching entry.
		if !errors.Is(apiError, errMCPToolNotFound) {
			return fmt.Errorf("unexpected error verifying resource of type %s with ID '%s' was destroyed: %v", resourceType, rs.Primary.ID, apiError)
		}
	}

	return nil
}

// ============================================
// ===         Test helper functions        ===
// ============================================

func getMCPToolCreateTime(s *terraform.State, name string) (float64, error) {
	resourceID, stateError := getResourceIDFromState(s, name)
	if stateError != nil {
		return 0, stateError
	}
	conf := testAccOpendistroProvider.Meta().(*ProviderConf)

	tool, err := getMCPToolFromAPI(context.Background(), conf, resourceID)
	if err != nil {
		return 0, err
	}

	createTime, ok := tool["create_time"].(float64)
	if !ok {
		return 0, fmt.Errorf("create_time not found in registry entry for '%s': %v", resourceID, tool)
	}
	return createTime, nil
}

func jsonEquivalent(a, b interface{}) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

// ============================================
// ===         Test configurations          ===
// ============================================

func testAccOpensearchMLMCPToolConfig_Minimal() string {
	return `
resource "opensearch_ml_mcp_tool" "minimal" {
  name = "tf_acc_minimal"
  type = "ListIndexTool"
}
`
}

func testAccOpensearchMLMCPToolConfig_WithOptionalAttributes() string {
	return `
resource "opensearch_ml_mcp_tool" "with_optional" {
  name        = "tf_acc_with_optional"
  type        = "ListIndexTool"
  description = "Lists indices"

  # Deliberately not PPLTool: OpenSearch 3.1.0-3.7.0 reject a PPLTool with no model_id when
  # loading it into MCP server memory ("PPL tool needs non blank model id"), so it cannot
  # exercise the parameters attribute across the supported range. ListIndexTool has no
  # required parameters and accepts arbitrary ones.
  parameters = {
    model_type = "FINETUNE"
    execute    = "false"
  }

  attributes = jsonencode({
    input_schema = {
      type = "object"
      properties = {
        question = {
          type        = "string"
          description = "The question in natural language"
        }
      }
      required = ["question"]
    }
  })
}
`
}

func testAccOpensearchMLMCPToolConfig_Update() string {
	return `
resource "opensearch_ml_mcp_tool" "update" {
  name        = "tf_acc_update"
  type        = "ListIndexTool"
  description = "Lists indices"

  attributes = jsonencode({
    input_schema = {
      type = "object"
      properties = {
        indices = {
          type  = "array"
          items = { type = "string" }
        }
      }
    }
  })
}
`
}

func testAccOpensearchMLMCPToolConfig_Update_Updated() string {
	return `
resource "opensearch_ml_mcp_tool" "update" {
  name        = "tf_acc_update"
  type        = "ListIndexTool"
  description = "Lists indices, with a local filter"

  attributes = jsonencode({
    input_schema = {
      type = "object"
      properties = {
        indices = {
          type  = "array"
          items = { type = "string" }
        }
        local = { type = "boolean" }
      }
    }
  })
}
`
}

func testAccOpensearchMLMCPToolConfig_ForceNew(toolType string) string {
	return fmt.Sprintf(`
resource "opensearch_ml_mcp_tool" "force_new" {
  name = "tf_acc_force_new"
  type = %q
}
`, toolType)
}

func testAccOpensearchMLMCPToolConfig_Normalization_Ordered() string {
	return `
resource "opensearch_ml_mcp_tool" "normalization" {
  name = "tf_acc_normalization"
  type = "SearchIndexTool"

  attributes = <<EOF
{
  "input_schema": {
    "type": "object",
    "properties": {
      "index": { "type": "string" },
      "query": { "type": "object" }
    },
    "required": ["index", "query"]
  }
}
EOF
}
`
}

func testAccOpensearchMLMCPToolConfig_Normalization_Reordered() string {
	return `
resource "opensearch_ml_mcp_tool" "normalization" {
  name = "tf_acc_normalization"
  type = "SearchIndexTool"

  attributes = <<EOF
{"input_schema":{"required":["index","query"],"properties":{"query":{"type":"object"},"index":{"type":"string"}},"type":"object"}}
EOF
}
`
}

func testAccOpensearchMLMCPToolConfig_InvalidAttributes() string {
	return `
resource "opensearch_ml_mcp_tool" "invalid_attributes" {
  name = "tf_acc_invalid_attributes"
  type = "ListIndexTool"

  # A JSON object with a trailing comma and an unquoted key: rejected by the schema's
  # ValidateFunc, so no request reaches the cluster.
  attributes = "{\"input_schema\": {type: \"object\"},}"
}
`
}

func testAccOpensearchMLMCPToolConfig_OutOfBand() string {
	return `
resource "opensearch_ml_mcp_tool" "out_of_band" {
  name = "tf_acc_out_of_band"
  type = "ListIndexTool"
}
`
}

func testAccOpensearchMLMCPToolConfig_Parallel() string {
	return `
resource "opensearch_ml_mcp_tool" "parallel_a" {
  name = "tf_acc_parallel_a"
  type = "ListIndexTool"
}

resource "opensearch_ml_mcp_tool" "parallel_b" {
  name = "tf_acc_parallel_b"
  type = "IndexMappingTool"
}

resource "opensearch_ml_mcp_tool" "parallel_c" {
  name = "tf_acc_parallel_c"
  type = "SearchIndexTool"
}

resource "opensearch_ml_mcp_tool" "parallel_d" {
  name = "tf_acc_parallel_d"
  type = "ListIndexTool"
}
`
}

func testAccOpensearchMLMCPToolConfig_FailedLoad() string {
	return `
resource "opensearch_ml_mcp_tool" "failed_load" {
  name = "tf_acc_failed_load"
  type = "PPLTool"
}
`
}
