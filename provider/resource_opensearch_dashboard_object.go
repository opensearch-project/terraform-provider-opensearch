package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

const (
	SECURITY_TENANT_HEADER = "securitytenant"
)

func resourceOpensearchDashboardObject() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Dashboards object resource. This resource interacts directly with the underlying OpenSearch index backing Dashboards, so the format must match what Dashboards the version of Dashboards is expecting. Dashboards with older versions - directly pulling the JSON from a Dashboards index of the same version of OpenSearch targeted by the provider is a workaround.",
		Create:      resourceOpensearchDashboardObjectCreate,
		Read:        resourceOpensearchDashboardObjectRead,
		Update:      resourceOpensearchDashboardObjectUpdate,
		Delete:      resourceOpensearchDashboardObjectDelete,
		CustomizeDiff: customdiff.ForceNewIfChange(
			"body",
			// force recreation if _id of object changed
			func(ctx context.Context, old, new, meta interface{}) bool {
				dashboardObjectOld, err := readBodyInterface(old)
				if err != nil {
					return false
				}
				dashboardObjectNew, err := readBodyInterface(new)
				if err != nil {
					return false
				}
				return !(dashboardObjectOld["_id"] == dashboardObjectNew["_id"])
			}),
		Schema: map[string]*schema.Schema{
			"body": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(i interface{}, k string) (warnings []string, errors []error) {
					v, ok := i.(string)
					if !ok {
						errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
						return warnings, errors
					}

					if _, err := structure.NormalizeJsonString(v); err != nil {
						errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
						return warnings, errors
					}

					var body []interface{}
					if err := json.Unmarshal([]byte(v), &body); err != nil {
						errors = append(errors, fmt.Errorf("%q must be an array of objects: %s", k, err))
						return warnings, errors
					}

					for _, o := range body {
						dashboardObject, ok := o.(map[string]interface{})
						if !ok {
							errors = append(errors, fmt.Errorf("entries must be objects"))
							continue
						}

						for _, k := range []string{"_source", "_id"} {
							if dashboardObject[k] == nil {
								errors = append(errors, fmt.Errorf("object must have the %q key", k))
							}
						}
					}
					return warnings, errors
				},
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				Description: "The JSON body of the dashboard object.",
			},
			"tenant_name": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Default:       "",
				Description:   "The name of the tenant to which dashboard data associate. Empty string defaults to global tenant.",
				ConflictsWith: []string{"index"},
			},
			"index": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Default:       "",
				Description:   "The name of the index where dashboard data is stored. Does not work with tenant_name.",
				ConflictsWith: []string{"tenant_name"},
			},
		},
	}
}

func resourceOpensearchDashboardObjectCreate(d *schema.ResourceData, meta interface{}) error {
	// parse desired terrafrom state
	state, err := readDashboardObjectState(d)
	if err != nil {
		return err
	}
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return fmt.Errorf("Could not read client: %+v", err)
	}

	// make OpenSearch API calls
	if err = createDashboardIndexIfNotExists(client, state.index); err != nil {
		return fmt.Errorf("Failed to create new Dashboard index: %+v", err)
	}
	resp, err := state.putDashboardObject(client)
	if err != nil {
		return fmt.Errorf("Failed to put Dashboard object: %+v", err)
	}

	// set computed value
	d.SetId(resp.Id)
	return resourceOpensearchDashboardObjectRead(d, meta)
}

func resourceOpensearchDashboardObjectRead(d *schema.ResourceData, meta interface{}) error {
	// parse current terraform state
	state, err := readDashboardObjectState(d)
	if err != nil {
		return err
	}
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	// fetch object from OpenSearch
	result, err := state.getDashboardObject(client)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] Dashboard Object (%s) not found, removing from state", state.id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Could not read state from OpenSearch: %+v", err)
	}

	// build json string from response that represents body configuration
	// Note that only the 'original' keys are considered. Keys that
	// OpenSearch adds internally will be ignored (e.g. 'updated_at').
	log.Printf("[TRACE] body source: %s", string(result.Source))

	var source map[string]interface{}
	if err := json.Unmarshal(result.Source, &source); err != nil {
		return fmt.Errorf("Failed to unmarshal source: %+v", err)
	}

	var originalKeys []string
	for k := range state.dashboardObject {
		originalKeys = append(originalKeys, k)
	}

	stateObject := []map[string]interface{}{make(map[string]interface{})}
	for _, k := range originalKeys {
		if k == "_id" {
			// _id is returned separately in the response, not in _source
			stateObject[0][k] = result.Id
		} else if k == "_source" {
			// The entire source is the _source content
			stateObject[0][k] = source
		} else {
			stateObject[0][k] = source[k]
		}
	}
	bodyBytes, err := json.Marshal(stateObject)
	if err != nil {
		return fmt.Errorf("Failed marshalling resource data: %+v", err)
	}

	// update terraform state based on fetched data. Fields other than 'body' do
	// not need to be updated as chanages in these fields result in 'NotFound'
	ds := &resourceDataSetter{d: d}
	ds.set("body", string(bodyBytes))

	return ds.err
}

func resourceOpensearchDashboardObjectUpdate(d *schema.ResourceData, meta interface{}) error {
	// parse desired terraform state
	state, err := readDashboardObjectState(d)
	if err != nil {
		return err
	}
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	// update data in OpenSearch via put request
	resp, err := state.putDashboardObject(client)
	if err != nil {
		return fmt.Errorf("Dashboard object update failed: %+v", err)
	}

	// update computed values
	d.SetId(resp.Id)
	return resourceOpensearchDashboardObjectRead(d, meta)
}

func resourceOpensearchDashboardObjectDelete(d *schema.ResourceData, meta interface{}) error {
	// read old values. note that readDashboardObjectState(d) would read new state
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	index, _ := d.GetChange("index")
	indexStr := index.(string)
	tenantName, _ := d.GetChange("tenant_name")
	tenantNameStr := tenantName.(string)

	// Calculate index if tenantName is given
	if tenantNameStr != "" {
		indexStr, err = resourceOpensearchOpenDistroDashboardComputeIndex(tenantNameStr)
		if err != nil {
			return fmt.Errorf("could not compute tenant name: %+v", err)
		}
	}

	// Default for index if still empty
	if indexStr == "" {
		indexStr = ".kibana"
	}

	// make delete api call
	return deleteDashboardObject(client, indexStr, d.Id(), tenantNameStr)
}

func createDashboardIndexIfNotExists(client *OpenSearchClient, index string) error {
	log.Printf("[INFO] createDashboardIndexIfNotExists %s", index)

	// Check if index exists
	existsReq := opensearchapi.IndicesExistsReq{
		Indices: []string{index},
	}
	existsResp, err := client.Client.Indices.Exists(context.TODO(), existsReq)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %+v", err)
	}

	if existsResp.StatusCode == http.StatusOK {
		// Index already exists
		return nil
	}

	// Create the index
	createReq := opensearchapi.IndicesCreateReq{
		Index: index,
		Body:  bytes.NewReader([]byte(`{"mappings":{}}`)),
	}
	createResp, err := client.Client.Indices.Create(context.TODO(), createReq)
	if err != nil {
		return fmt.Errorf("failed to create OpenSearch index: %+v", err)
	}

	if createResp.Acknowledged {
		log.Printf("[INFO] Created new Dashboard index")
		return nil
	}

	return fmt.Errorf("failed to create OpenSearch index: index creation not acknowledged")
}

type dashboardObjectState struct {
	index      string
	tenantName string
	// body splitted into interfaces
	dashboardObject map[string]interface{}
	// id from body in dashboard object resource
	id string
}

func readDashboardObjectState(d *schema.ResourceData) (*dashboardObjectState, error) {
	dashboardObject, err := readBodyInterface(d.Get("body"))
	if err != nil {
		return nil, fmt.Errorf("Could not read body interface: %+v", err)
	}
	// Calculate index if tenantName is given
	indexName := d.Get("index").(string)
	tenantName := d.Get("tenant_name").(string)
	if tenantName != "" {
		indexName, err = resourceOpensearchOpenDistroDashboardComputeIndex(tenantName)
		if err != nil {
			return nil, fmt.Errorf("could not compute tenant name: %+v", err)
		}
	}
	// Default to .kibana
	if indexName == "" {
		indexName = ".kibana"
	}
	return &dashboardObjectState{
		index:           indexName,
		tenantName:      tenantName,
		dashboardObject: dashboardObject,
		id:              dashboardObject["_id"].(string),
	}, nil
}

func readBodyInterface(i interface{}) (map[string]interface{}, error) {
	bodyString, ok := i.(string)
	if !ok {
		return nil, fmt.Errorf("Cannot convert input to string.")
	}

	var body []interface{}
	if err := json.Unmarshal([]byte(bodyString), &body); err != nil {
		return nil, fmt.Errorf("Could not unmarshal body string: %+v", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("Body has no elements as JSON array.")
	}

	dashboardObject, ok := body[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Body has unexpected format.")
	}

	return dashboardObject, nil
}

type dashboardObjectResponse struct {
	Id     string `json:"_id"`
	Index  string `json:"_index"`
	Result string `json:"result"`
}

func (s *dashboardObjectState) putDashboardObject(client *OpenSearchClient) (*dashboardObjectResponse, error) {
	// Convert source to JSON
	sourceJSON, err := json.Marshal(s.dashboardObject["_source"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal source: %+v", err)
	}

	// Build the path
	path := fmt.Sprintf("/%s/_doc/%s", s.index, s.id)

	// Create request
	req, err := http.NewRequest("PUT", client.config.rawUrl+path, bytes.NewReader(sourceJSON))
	if err != nil {
		return nil, fmt.Errorf("error building PUT request: %+v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add tenant header if needed
	if s.tenantName != "" {
		req.Header.Set(SECURITY_TENANT_HEADER, s.tenantName)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("error putting dashboard object: %+v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %+v", err)
	}

	// Check status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("error putting dashboard object: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result dashboardObjectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %+v", err)
	}

	// Ensure ID is set
	result.Id = s.id

	return &result, nil
}

type getResponse struct {
	Found   bool            `json:"found"`
	Source  json.RawMessage `json:"_source"`
	Id      string          `json:"_id"`
	Index   string          `json:"_index"`
	Version int64           `json:"_version"`
}

func (s *dashboardObjectState) getDashboardObject(client *OpenSearchClient) (*getResponse, error) {
	// Build the path
	path := fmt.Sprintf("/%s/_doc/%s", s.index, s.id)

	// Create request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return nil, fmt.Errorf("error building GET request: %+v", err)
	}

	// Add tenant header if needed
	if s.tenantName != "" {
		req.Header.Set(SECURITY_TENANT_HEADER, s.tenantName)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("error getting dashboard object: %+v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %+v", err)
	}

	// Check for not found
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("dashboard object not found: %s/%s", s.index, s.id)
	}

	// Check status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("error getting dashboard object: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result getResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %+v", err)
	}

	return &result, nil
}

func deleteDashboardObject(client *OpenSearchClient, index, id, tenantName string) error {
	// Build the path
	path := fmt.Sprintf("/%s/_doc/%s", index, id)

	// Create request
	req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	if err != nil {
		return fmt.Errorf("error building DELETE request: %+v", err)
	}

	// Add tenant header if needed
	if tenantName != "" {
		req.Header.Set(SECURITY_TENANT_HEADER, tenantName)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return fmt.Errorf("error deleting dashboard object: %+v", err)
	}
	defer resp.Body.Close()

	// Read response (needed to consume body)
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %+v", err)
	}

	// We'll get an error if it's not found (404), which is acceptable in delete
	// The calling code checks for other errors
	return nil
}
