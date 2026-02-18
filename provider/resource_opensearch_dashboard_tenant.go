package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var openSearchDashboardTenantSchema = map[string]*schema.Schema{
	"tenant_name": {
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
		Description: "The name of the tenant.",
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Description of the tenant.",
	},
	"index": {
		Type:     schema.TypeString,
		Computed: true,
	},
}

func resourceOpenSearchDashboardTenant() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchOpenDistroDashboardTenantCreate,
		Read:   resourceOpensearchOpenDistroDashboardTenantRead,
		Update: resourceOpensearchOpenDistroDashboardTenantUpdate,
		Delete: resourceOpensearchOpenDistroDashboardTenantDelete,
		Schema: openSearchDashboardTenantSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Provides an OpenSearch dashboard tenant resource. Please refer to the OpenSearch documentation for details.",
	}
}

func resourceOpensearchOpenDistroDashboardTenantCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroDashboardTenant(d, m); err != nil {
		log.Printf("[INFO] Failed to create OpenDistroDashboardTenant: %+v", err)
		return err
	}

	name := d.Get("tenant_name").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroDashboardTenantRead(d, m)
}

func resourceOpensearchOpenDistroDashboardTenantRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroDashboardTenant(d.Id(), m)

	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] OpenDistroDashboardTenant (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("tenant_name", d.Id()); err != nil {
		return fmt.Errorf("error setting tenant_name: %s", err)
	}
	if err := d.Set("description", res.Description); err != nil {
		return fmt.Errorf("error setting description: %s", err)
	}

	index, err := resourceOpensearchOpenDistroDashboardComputeIndex(d.Id())
	if err != nil {
		return err
	}
	if err := d.Set("index", index); err != nil {
		return fmt.Errorf("error setting index: %s", err)
	}

	return nil
}

func resourceOpensearchOpenDistroDashboardComputeIndex(tenant string) (string, error) {
	// Calc Hash
	hashSum := int32(0)
	for _, char := range tenant {
		shift := (hashSum << 5)
		hashSum = (shift - hashSum) + int32(char-0)
	}
	// remove all chars that are not alphanumeric
	alphanumeric, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "", err
	}
	cleanedTenant := alphanumeric.ReplaceAllString(tenant, "")

	// originalDashboardIndex+"_"+tenant.hashCode()+"_"+tenant.toLowerCase().replaceAll("[^a-z0-9]+", "")
	return fmt.Sprintf(".kibana_%v_%v", hashSum, strings.ToLower(cleanedTenant)), nil
}

func resourceOpensearchOpenDistroDashboardTenantUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroDashboardTenant(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroDashboardTenantRead(d, m)
}

func resourceOpensearchOpenDistroDashboardTenantDelete(d *schema.ResourceData, m interface{}) error {
	path := fmt.Sprintf("/_plugins/_security/api/tenants/%s", d.Get("tenant_name").(string))

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
		if err != nil {
			return fmt.Errorf("error building DELETE request: %w", err)
		}

		resp, err = client.Client.Client.Perform(req)
		if err == nil {
			// Check if we should retry on conflict or internal server error
			if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
				break
			}
			resp.Body.Close()
		}
	}

	if err != nil {
		return fmt.Errorf("error deleting tenant: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting tenant: received status code %d", resp.StatusCode)
}

func resourceOpensearchGetOpenDistroDashboardTenant(tenantID string, m interface{}) (TenantBody, error) {
	var err error
	tenant := new(TenantBody)

	path := fmt.Sprintf("/_plugins/_security/api/tenants/%s", tenantID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return *tenant, err
	}

	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return *tenant, fmt.Errorf("error building GET request: %w", err)
	}

	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return *tenant, fmt.Errorf("error getting tenant: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return *tenant, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return *tenant, fmt.Errorf("tenant not found: %s", tenantID)
	}

	if resp.StatusCode != http.StatusOK {
		return *tenant, fmt.Errorf("error getting tenant: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	var tenantDefinition map[string]TenantBody

	if err := json.Unmarshal(body, &tenantDefinition); err != nil {
		return *tenant, fmt.Errorf("error unmarshalling tenant body: %+v: %+v", err, body)
	}

	*tenant = tenantDefinition[tenantID]

	return *tenant, nil
}

func resourceOpensearchPutOpenDistroDashboardTenant(d *schema.ResourceData, m interface{}) (*TenantResponse, error) {
	response := new(TenantResponse)

	tenantsDefinition := TenantBody{
		Description: d.Get("description").(string),
	}

	tenantJSON, err := json.Marshal(tenantsDefinition)
	if err != nil {
		return response, fmt.Errorf("Body Error : %s", tenantJSON)
	}

	path := fmt.Sprintf("/_plugins/_security/api/tenants/%s", d.Get("tenant_name").(string))

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Execute request with retry logic
	var resp *http.Response
	var body []byte
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(string(tenantJSON)))
		if err != nil {
			return response, fmt.Errorf("error building PUT request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Client.Client.Perform(req)
		if err == nil {
			// Read response body
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return response, fmt.Errorf("error reading response body: %w", err)
			}

			// Check if we should retry on conflict or internal server error
			if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
				break
			}
		}
	}

	if err != nil {
		return response, fmt.Errorf("error putting tenant: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error putting tenant: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling tenant body: %+v: %+v", err, body)
	}

	return response, nil
}

type TenantResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type TenantBody struct {
	Description string `json:"description"`
}
