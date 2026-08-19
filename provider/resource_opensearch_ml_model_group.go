package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpensearchMLModelGroup() *schema.Resource {
	return &schema.Resource{
		Description:   "OpenSearch ML Model Group resource",
		CreateContext: resourceOpensearchMLModelGroupCreate,
		ReadContext:   resourceOpensearchMLModelGroupRead,
		UpdateContext: resourceOpensearchMLModelGroupUpdate,
		DeleteContext: resourceOpensearchMLModelGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// ============================================
			// ===         Required attributes          ===
			// ============================================
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the ML Model Group. Must be unique within the cluster.",
			},

			// ============================================
			// ===         Optional attributes          ===
			// ============================================
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the ML Model Group",
			},
			"access_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Valid values are `\"public\"`, `\"private\"`, and `\"restricted\"`. When `\"restricted\"`, `backend_roles` or `add_all_backend_roles` must be set, but not both. If none of the security parameters (`access_mode`, `backend_roles`, and `add_all_backend_roles`) are set, the default `access_mode` is `\"private\"`.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diff when config is empty and API returns 'private' (the default)
					return new == "" && old == "private"
				},
			},
			"backend_roles": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Description:   "List of the ML Model Group owner’s OpenSearch backend roles to add to the ML Model Group. Can be specified only if `access_mode` is `\"restricted\"`. Conflicts with `add_all_backend_roles`.",
				ConflictsWith: []string{"add_all_backend_roles"},
			},
			"add_all_backend_roles": {
				Type:          schema.TypeBool,
				Optional:      true,
				Description:   "If `true`, all OpenSearch  backend roles of the ML Model owner are added to the ML Model Group. Can be specified only if `access_mode` is `\"restricted\"`. Conflicts with `backend_roles`. Admin users cannot set this to `true`.",
				ConflictsWith: []string{"backend_roles"},
			},
		},
	}
}

func resourceOpensearchMLModelGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload := buildMLModelGroupPayload(d)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal ML Model Group payload: %s", err)
	}

	url := conf.rawUrl + "/_plugins/_ml/model_groups/_register"
	result, err := performRequestAndParse(ctx, conf.osClient, "POST", url, strings.NewReader(string(jsonPayload)), "register ML Model Group")
	if err != nil {
		return diag.FromErr(err)
	}

	modelGroupID, ok := result["model_group_id"].(string)
	if !ok || modelGroupID == "" {
		return diag.Errorf("model_group_id not found in response: %v", result)
	}

	d.SetId(modelGroupID)
	return resourceOpensearchMLModelGroupRead(ctx, d, m)
}

func resourceOpensearchMLModelGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	modelGroup, err := getMLModelGroupFromAPI(ctx, conf, d.Id())
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if name, ok := modelGroup["name"].(string); ok {
		if err := d.Set("name", name); err != nil {
			return diag.Errorf("error setting name: %s", err)
		}
	}

	if desc, ok := modelGroup["description"].(string); ok {
		if err := d.Set("description", desc); err != nil {
			return diag.Errorf("error setting description: %s", err)
		}
	}

	// API returns 'access' field, map it to 'access_mode' in state
	if access, ok := modelGroup["access"].(string); ok {
		access = strings.ToLower(access)
		if err := d.Set("access_mode", access); err != nil {
			return diag.Errorf("error setting access_mode: %s", err)
		}
	}

	if backendRoles, ok := modelGroup["backend_roles"].([]interface{}); ok {
		if err := d.Set("backend_roles", backendRoles); err != nil {
			return diag.Errorf("error setting backend_roles: %s", err)
		}
	}

	if addAllBackendRoles, ok := modelGroup["add_all_backend_roles"].(bool); ok {
		if err := d.Set("add_all_backend_roles", addAllBackendRoles); err != nil {
			return diag.Errorf("error setting add_all_backend_roles: %s", err)
		}
	}

	return nil
}

func resourceOpensearchMLModelGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload := buildMLModelGroupPayload(d)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal ML Model Group update payload: %s", err)
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/model_groups/%s", d.Id())
	if _, err := performRequestAndParse(ctx, conf.osClient, "PUT", url, strings.NewReader(string(jsonPayload)), "update ML Model Group"); err != nil {
		return diag.FromErr(err)
	}

	return resourceOpensearchMLModelGroupRead(ctx, d, m)
}

func resourceOpensearchMLModelGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/model_groups/%s", d.Id())
	_, err := performRequestAndParse(ctx, conf.osClient, "DELETE", url, nil, "delete ML Model Group")
	if err != nil {
		var httpErr *HTTPError
		// Ignore 404 errors - resource is already deleted
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

// ============================================
// ===         Helper functions             ===
// ============================================

func buildMLModelGroupPayload(d *schema.ResourceData) map[string]interface{} {
	payload := map[string]interface{}{
		"name": d.Get("name").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		payload["description"] = v.(string)
	}
	if v, ok := d.GetOk("access_mode"); ok {
		payload["access_mode"] = v.(string)
	}
	if v, ok := d.GetOk("backend_roles"); ok {
		payload["backend_roles"] = v.([]interface{})
	}
	if v, ok := d.GetOk("add_all_backend_roles"); ok {
		payload["add_all_backend_roles"] = v.(bool)
	}

	return payload
}

func getMLModelGroupFromAPI(ctx context.Context, conf *ProviderConf, modelGroupID string) (map[string]interface{}, error) {
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/model_groups/%s", modelGroupID)
	return performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "get ML Model Group")
}
