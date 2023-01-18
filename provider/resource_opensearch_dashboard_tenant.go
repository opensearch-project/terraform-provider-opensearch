package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var openSearchDashboardTenantSchema = map[string]*schema.Schema{
	"tenant_name": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"description": {
		Type:     schema.TypeString,
		Optional: true,
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
		if elastic7.IsNotFound(err) {
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
	return fmt.Sprintf(".dashboard_%v_%v", hashSum, strings.ToLower(cleanedTenant)), nil
}

func resourceOpensearchOpenDistroDashboardTenantUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroDashboardTenant(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroDashboardTenantRead(d, m)
}

func resourceOpensearchOpenDistroDashboardTenantDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_opendistro/_security/api/tenants/{name}", map[string]string{
		"name": d.Get("tenant_name").(string),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for tenant: %+v", err)
	}

	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           "DELETE",
		Path:             path,
		RetryStatusCodes: []int{http.StatusConflict, http.StatusInternalServerError},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})

	return err
}

func resourceOpensearchGetOpenDistroDashboardTenant(tenantID string, m interface{}) (TenantBody, error) {
	var err error
	tenant := new(TenantBody)

	path, err := uritemplates.Expand("/_opendistro/_security/api/tenants/{name}", map[string]string{
		"name": tenantID,
	})

	if err != nil {
		return *tenant, fmt.Errorf("error building URL path for tenant: %+v", err)
	}

	var body json.RawMessage
	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return *tenant, err
	}
	var res *elastic7.Response
	res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return *tenant, err
	}
	body = res.Body

	var tenantDefinition map[string]TenantBody

	if err := json.Unmarshal(body, &tenantDefinition); err != nil {
		return *tenant, fmt.Errorf("error unmarshalling tenant body: %+v: %+v", err, body)
	}

	*tenant = tenantDefinition[tenantID]

	return *tenant, err
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

	path, err := uritemplates.Expand("/_opendistro/_security/api/tenants/{name}", map[string]string{
		"name": d.Get("tenant_name").(string),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for tenant: %+v", err)
	}

	var body json.RawMessage
	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           "PUT",
		Path:             path,
		Body:             string(tenantJSON),
		RetryStatusCodes: []int{http.StatusConflict, http.StatusInternalServerError},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

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
