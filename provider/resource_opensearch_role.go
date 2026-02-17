package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var openDistroRoleSchema = map[string]*schema.Schema{
	"role_name": {
		Description: "The name of the security role.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"cluster_permissions": {
		Description: "A list of cluster permissions.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"index_permissions": {
		Description: "A configuration of index permissions",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"index_patterns": {
					Description: "A list of glob patterns for the index names.",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Set: schema.HashString,
				},
				"document_level_security": {
					Description: "A selector for document-level security (json formatted using jsonencode).",
					Type:        schema.TypeString,
					Optional:    true,
				},
				"field_level_security": {
					Description: "A list of selectors for field-level security.",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Set: schema.HashString,
				},
				"masked_fields": {
					Description: "A list of masked fields",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Set: schema.HashString,
				},
				"allowed_actions": {
					Description: "A list of allowed actions.",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Set: schema.HashString,
				},
			},
		},
		Set: indexPermissionsHash,
	},
	"tenant_permissions": {
		Description: "A configuration of tenant permissions",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"tenant_patterns": {
					Description: "A list of glob patterns for the tenant names",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Set: schema.HashString,
				},
				"allowed_actions": {
					Description: "A list of allowed actions.",
					Type:        schema.TypeSet,
					Optional:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Set: schema.HashString,
				},
			},
		},
		Set: tenantPermissionsHash,
	},
	"description": {
		Description: "Description of the role.",
		Type:        schema.TypeString,
		Optional:    true,
	},
}

func resourceOpenSearchRole() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch security role resource. Please refer to the OpenSearch Access Control documentation for details.",
		Create:      resourceOpensearchOpenDistroRoleCreate,
		Read:        resourceOpensearchOpenDistroRoleRead,
		Update:      resourceOpensearchOpenDistroRoleUpdate,
		Delete:      resourceOpensearchOpenDistroRoleDelete,
		Schema:      openDistroRoleSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchOpenDistroRoleCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroRole(d, m); err != nil {
		log.Printf("[INFO] Failed to create OpenDistroRole: %+v", err)
		return err
	}

	name := d.Get("role_name").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroRoleRead(d, m)
}

func resourceOpensearchOpenDistroRoleRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroRole(d.Id(), m)

	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] OpenDistroRole (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("role_name", d.Id()); err != nil {
		return fmt.Errorf("error setting role_name: %s", err)
	}
	if err := d.Set("tenant_permissions", flattenTenantPermissions(res.TenantPermissions)); err != nil {
		return fmt.Errorf("error setting tenant_permissions: %s", err)
	}
	if err := d.Set("cluster_permissions", res.ClusterPermissions); err != nil {
		return fmt.Errorf("error setting cluster_permissions: %s", err)
	}
	if err := d.Set("index_permissions", flattenIndexPermissions(res.IndexPermissions, d)); err != nil {
		return fmt.Errorf("error setting index_permissions: %s", err)
	}
	if err := d.Set("description", res.Description); err != nil {
		return fmt.Errorf("error setting description: %s", err)
	}

	return nil
}

func resourceOpensearchOpenDistroRoleUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroRole(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroRoleRead(d, m)
}

func resourceOpensearchOpenDistroRoleDelete(d *schema.ResourceData, m interface{}) error {
	roleName := d.Get("role_name").(string)
	path := fmt.Sprintf("/_plugins/_security/api/roles/%s", roleName)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	if err != nil {
		return fmt.Errorf("error building DELETE request: %w", err)
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		resp, err = client.Client.Client.Perform(req)
		if err == nil && resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if err != nil {
		return fmt.Errorf("error deleting role: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting role: received status code %d", resp.StatusCode)
}

func resourceOpensearchGetOpenDistroRole(roleID string, m interface{}) (RoleBody, error) {
	var err error
	role := new(RoleBody)
	path := fmt.Sprintf("/_plugins/_security/api/roles/%s", roleID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return *role, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return *role, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return *role, fmt.Errorf("error getting role: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return *role, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return *role, fmt.Errorf("role not found: %s", roleID)
	}

	if resp.StatusCode != http.StatusOK {
		return *role, fmt.Errorf("error getting role: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	var roleDefinition map[string]RoleBody

	if err := json.Unmarshal(body, &roleDefinition); err != nil {
		return *role, fmt.Errorf("error unmarshalling role body: %+v: %+v", err, body)
	}

	*role = roleDefinition[roleID]

	return *role, err
}

func resourceOpensearchPutOpenDistroRole(d *schema.ResourceData, m interface{}) (*RoleResponse, error) {
	response := new(RoleResponse)

	indexPermissions, err := expandIndexPermissionsSet(d.Get("index_permissions").(*schema.Set).List())
	if err != nil {
		fmt.Print("Error in index get : ", err)
	}
	var indexPermissionsBody []IndexPermissions
	for _, idx := range indexPermissions {
		putIdx := IndexPermissions{
			IndexPatterns:         idx.IndexPatterns,
			DocumentLevelSecurity: idx.DocumentLevelSecurity,
			FieldLevelSecurity:    idx.FieldLevelSecurity,
			MaskedFields:          idx.MaskedFields,
			AllowedActions:        idx.AllowedActions,
		}
		indexPermissionsBody = append(indexPermissionsBody, putIdx)
	}

	tenantPermissions, err := expandTenantPermissionsSet(d.Get("tenant_permissions").(*schema.Set).List())
	if err != nil {
		fmt.Print("Error in tenant get : ", err)
	}
	var tenantPermissionsBody []TenantPermissions
	for _, tenant := range tenantPermissions {
		putTeanant := TenantPermissions{
			TenantPatterns: tenant.TenantPatterns,
			AllowedActions: tenant.AllowedActions,
		}
		tenantPermissionsBody = append(tenantPermissionsBody, putTeanant)
	}

	rolesDefinition := RoleBody{
		ClusterPermissions: expandStringList(d.Get("cluster_permissions").(*schema.Set).List()),
		IndexPermissions:   indexPermissionsBody,
		TenantPermissions:  tenantPermissionsBody,
		Description:        d.Get("description").(string),
	}

	roleJSON, err := json.Marshal(rolesDefinition)
	if err != nil {
		return response, fmt.Errorf("Body Error : %s", roleJSON)
	}

	roleName := d.Get("role_name").(string)
	path := fmt.Sprintf("/_plugins/_security/api/roles/%s", roleName)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(string(roleJSON)))
	if err != nil {
		return response, fmt.Errorf("error building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request with retry logic
	// see https://github.com/opendistro-for-elasticsearch/security/issues/1095
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		resp, err = client.Client.Client.Perform(req)
		if err == nil {
			// Check if we should retry
			if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
				break
			}
			resp.Body.Close()
		} else {
			// Request failed, will retry
			if attempt < maxRetries-1 {
				continue
			}
		}
	}

	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error creating role: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling role body: %+v: %+v", err, body)
	}

	return response, nil
}

type RoleResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type RoleBody struct {
	Description        string              `json:"description"`
	ClusterPermissions []string            `json:"cluster_permissions,omitempty"`
	IndexPermissions   []IndexPermissions  `json:"index_permissions,omitempty"`
	TenantPermissions  []TenantPermissions `json:"tenant_permissions,omitempty"`
}

type IndexPermissions struct {
	IndexPatterns         []string `json:"index_patterns"`
	DocumentLevelSecurity string   `json:"dls"`
	FieldLevelSecurity    []string `json:"fls"`
	MaskedFields          []string `json:"masked_fields"`
	AllowedActions        []string `json:"allowed_actions"`
}

type TenantPermissions struct {
	TenantPatterns []string `json:"tenant_patterns"`
	AllowedActions []string `json:"allowed_actions"`
}
