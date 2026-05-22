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

var openDistroRolesMappingSchema = map[string]*schema.Schema{
	"role_name": {
		Description: "The name of the security role.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"backend_roles": {
		Description: "A list of backend roles.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"hosts": {
		Description: "A list of host names.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"users": {
		Description: "A list of users.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"description": {
		Description: "Description of the role mapping.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"and_backend_roles": {
		Description: "A list of backend roles.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
}

func resourceOpenSearchRolesMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchOpenDistroRolesMappingCreate,
		Read:   resourceOpensearchOpenDistroRolesMappingRead,
		Update: resourceOpensearchOpenDistroRolesMappingUpdate,
		Delete: resourceOpensearchOpenDistroRolesMappingDelete,
		Schema: openDistroRolesMappingSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Provides an OpenSearch security role mapping. Please refer to the OpenSearch Access Control documentation for details.",
	}
}

func resourceOpensearchOpenDistroRolesMappingCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroRolesMapping(d, m); err != nil {
		log.Printf("[INFO] Failed to put role mapping: %+v", err)
		return err
	}

	name := d.Get("role_name").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroRolesMappingRead(d, m)
}

func resourceOpensearchOpenDistroRolesMappingRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroRolesMapping(d.Id(), m)

	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] OpenDistroRolesMapping (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("role_name", d.Id()); err != nil {
		return fmt.Errorf("error setting role_name: %s", err)
	}
	if err := d.Set("backend_roles", res.BackendRoles); err != nil {
		return fmt.Errorf("error setting backend_roles: %s", err)
	}
	if err := d.Set("hosts", res.Hosts); err != nil {
		return fmt.Errorf("error setting hosts: %s", err)
	}
	if err := d.Set("users", res.Users); err != nil {
		return fmt.Errorf("error setting users: %s", err)
	}
	if err := d.Set("description", res.Description); err != nil {
		return fmt.Errorf("error setting description: %s", err)
	}
	if err := d.Set("and_backend_roles", res.AndBackendRoles); err != nil {
		return fmt.Errorf("error setting and_backend_roles: %s", err)
	}

	return nil
}

func resourceOpensearchOpenDistroRolesMappingUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroRolesMapping(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroRolesMappingRead(d, m)
}

func resourceOpensearchOpenDistroRolesMappingDelete(d *schema.ResourceData, m interface{}) error {
	roleName := d.Get("role_name").(string)
	path := fmt.Sprintf("/_plugins/_security/api/rolesmapping/%s", roleName)

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
		return fmt.Errorf("error deleting role mapping: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting role mapping: received status code %d", resp.StatusCode)
}

func resourceOpensearchGetOpenDistroRolesMapping(roleID string, m interface{}) (RolesMapping, error) {
	var err error
	var roleMapping = new(RolesMapping)
	path := fmt.Sprintf("/_plugins/_security/api/rolesmapping/%s", roleID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return *roleMapping, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return *roleMapping, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return *roleMapping, fmt.Errorf("error getting role mapping: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return *roleMapping, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return *roleMapping, fmt.Errorf("role mapping not found: %s", roleID)
	}

	if resp.StatusCode != http.StatusOK {
		return *roleMapping, fmt.Errorf("error getting role mapping: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	var rolesMappingDefinition map[string]RolesMapping

	if err := json.Unmarshal(body, &rolesMappingDefinition); err != nil {
		return *roleMapping, fmt.Errorf("error unmarshalling role mapping body: %+v: %+v", err, body)
	}

	*roleMapping = rolesMappingDefinition[roleID]

	return *roleMapping, err
}

func resourceOpensearchPutOpenDistroRolesMapping(d *schema.ResourceData, m interface{}) (*RoleMappingResponse, error) {
	var err error
	response := new(RoleMappingResponse)

	rolesMappingDefinition := RolesMapping{
		BackendRoles:    expandStringList(d.Get("backend_roles").(*schema.Set).List()),
		Hosts:           expandStringList(d.Get("hosts").(*schema.Set).List()),
		Users:           expandStringList(d.Get("users").(*schema.Set).List()),
		Description:     d.Get("description").(string),
		AndBackendRoles: expandStringList(d.Get("and_backend_roles").(*schema.Set).List()),
	}
	roleJSON, err := json.Marshal(rolesMappingDefinition)

	if err != nil {
		return response, fmt.Errorf("Body Error : %s", roleJSON)
	}

	roleName := d.Get("role_name").(string)
	path := fmt.Sprintf("/_plugins/_security/api/rolesmapping/%s", roleName)

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
		return response, fmt.Errorf("error creating role mapping: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling role mapping body: %+v: %+v", err, body)
	}

	return response, nil
}

type RoleMappingResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type RolesMapping struct {
	BackendRoles    []string `json:"backend_roles"`
	Hosts           []string `json:"hosts"`
	Users           []string `json:"users"`
	Description     string   `json:"description"`
	AndBackendRoles []string `json:"and_backend_roles"`
}
