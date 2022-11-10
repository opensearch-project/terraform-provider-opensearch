package provider

import (
	"context"
	"encoding/json"
	"errors"
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

var openDistroKibanaTenantSchema = map[string]*schema.Schema{
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

func resourceOpenSearchKibanaTenant() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchOpenDistroKibanaTenantCreate,
		Read:   resourceOpensearchOpenDistroKibanaTenantRead,
		Update: resourceOpensearchOpenDistroKibanaTenantUpdate,
		Delete: resourceOpensearchOpenDistroKibanaTenantDelete,
		Schema: openDistroKibanaTenantSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchOpenDistroKibanaTenantCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroKibanaTenant(d, m); err != nil {
		log.Printf("[INFO] Failed to create OpenDistroKibanaTenant: %+v", err)
		return err
	}

	name := d.Get("tenant_name").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroKibanaTenantRead(d, m)
}

func resourceOpensearchOpenDistroKibanaTenantRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroKibanaTenant(d.Id(), m)

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] OpenDistroKibanaTenant (%s) not found, removing from state", d.Id())
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

	index, err := resourceOpensearchOpenDistroKibanaComputeIndex(d.Id())
	if err != nil {
		return err
	}
	if err := d.Set("index", index); err != nil {
		return fmt.Errorf("error setting index: %s", err)
	}

	return nil
}

func resourceOpensearchOpenDistroKibanaComputeIndex(tenant string) (string, error) {
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

	// originalKibanaIndex+"_"+tenant.hashCode()+"_"+tenant.toLowerCase().replaceAll("[^a-z0-9]+", "")
	return fmt.Sprintf(".kibana_%v_%v", hashSum, strings.ToLower(cleanedTenant)), nil
}

func resourceOpensearchOpenDistroKibanaTenantUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroKibanaTenant(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroKibanaTenantRead(d, m)
}

func resourceOpensearchOpenDistroKibanaTenantDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_opendistro/_security/api/tenants/{name}", map[string]string{
		"name": d.Get("tenant_name").(string),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for tenant: %+v", err)
	}

	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
			Method:           "DELETE",
			Path:             path,
			RetryStatusCodes: []int{http.StatusConflict, http.StatusInternalServerError},
			Retrier: elastic7.NewBackoffRetrier(
				elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
			),
		})
	default:
		err = errors.New("Creating tenants requires elastic v7 client")
	}

	return err
}

func resourceOpensearchGetOpenDistroKibanaTenant(tenantID string, m interface{}) (TenantBody, error) {
	var err error
	tenant := new(TenantBody)

	path, err := uritemplates.Expand("/_opendistro/_security/api/tenants/{name}", map[string]string{
		"name": tenantID,
	})

	if err != nil {
		return *tenant, fmt.Errorf("error building URL path for tenant: %+v", err)
	}

	var body json.RawMessage
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return *tenant, err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		var res *elastic7.Response
		res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
			Method: "GET",
			Path:   path,
		})
		if err != nil {
			return *tenant, err
		}
		body = res.Body
	default:
		return *tenant, errors.New("Creating tenants requires elastic v7 client")
	}

	var tenantDefinition map[string]TenantBody

	if err := json.Unmarshal(body, &tenantDefinition); err != nil {
		return *tenant, fmt.Errorf("error unmarshalling tenant body: %+v: %+v", err, body)
	}

	*tenant = tenantDefinition[tenantID]

	return *tenant, err
}

func resourceOpensearchPutOpenDistroKibanaTenant(d *schema.ResourceData, m interface{}) (*TenantResponse, error) {
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
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
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
	default:
		return response, errors.New("Creating tenants requires elastic v7 client")
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
