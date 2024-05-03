package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"

	elastic7 "github.com/olivere/elastic/v7"
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
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "",
				Description: "The name of the tenant to which dashboard data associate. Empty string defaults to global tenant.",
			},
			"index": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     ".kibana",
				Description: "The name of the index where dashboard data is stored. Does not work with tenant_name.",
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
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return fmt.Errorf("Could not read client: %+v", err)
	}

	// make OpenSearch API calls
	if err = elastic7CreateIndexIfNotExists(client, state.index); err != nil {
		return fmt.Errorf("Failed to create new Dashboard index: %+v", err)
	}
	resp, err := state.elastic7PutDashboardObject(client)
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
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	// fetch object from OpenSearch
	result, err := state.elastic7GetDashboardObject(client)
	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] Dashboard Object (%s) not found, removing from state", state.id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Could not read state from OpenSearch: %+v", err)
	}

	// build json string from response that represents body configuration
	// Note that only the 'original' keys are considered. Keys that
	// OpenSearch adds internally will be ignored (e.g. 'updated_at').
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("Failed to marshal result: %+v", err)
	}
	log.Printf("[TRACE] body: %s", string(resultJSON))

	var originalKeys []string
	for k := range state.dashboardObject {
		originalKeys = append(originalKeys, k)
	}

	res := make(map[string]interface{})
	if err := json.Unmarshal(resultJSON, &res); err != nil {
		return fmt.Errorf("Failed to unmarshal '%+v': %+v", resultJSON, err)
	}

	stateObject := []map[string]interface{}{make(map[string]interface{})}
	for _, k := range originalKeys {
		stateObject[0][k] = res[k]
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
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	// update data in OpenSearch via put request
	resp, err := state.elastic7PutDashboardObject(client)
	if err != nil {
		return fmt.Errorf("Dashboard object update failed: %+v", err)
	}

	// update computed values
	d.SetId(resp.Id)
	return resourceOpensearchDashboardObjectRead(d, meta)
}

func resourceOpensearchDashboardObjectDelete(d *schema.ResourceData, meta interface{}) error {
	// read old values. note that readDashboardObjectState(d) would read new state
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	index, _ := d.GetChange("index")
	tenantName, _ := d.GetChange("tenant_name")

	// make delete api call
	return elastic7DeleteDashboardObject(client, index.(string), d.Id(), tenantName.(string))
}

func elastic7CreateIndexIfNotExists(client *elastic7.Client, index string) error {
	log.Printf("[INFO] elastic7CreateIndexIfNotExists %s", index)
	exists, err := client.IndexExists(index).Do(context.TODO())
	if err != nil {
		return fmt.Errorf("%+v", err)
	}
	if !exists {
		createIndex, err := client.CreateIndex(index).Body(`{"mappings":{}}`).Do(context.TODO())
		if createIndex.Acknowledged {
			log.Printf("[INFO] Created new Dashboard index")
			return err
		}
		return fmt.Errorf("Failed to create OpenSearchsearch index: %+v", err)
	}
	return nil
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

	return &dashboardObjectState{
		index:           d.Get("index").(string),
		tenantName:      d.Get("tenant_name").(string),
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

func (s *dashboardObjectState) elastic7PutDashboardObject(client *elastic7.Client) (*elastic7.IndexResponse, error) {
	req := client.Index().Index(s.index).Id(s.id).BodyJson(s.dashboardObject["_source"])
	if s.tenantName != "" {
		req = req.Header(SECURITY_TENANT_HEADER, s.tenantName)
	}
	return req.Do(context.TODO())
}

func (s *dashboardObjectState) elastic7GetDashboardObject(client *elastic7.Client) (*elastic7.GetResult, error) {
	req := client.Get().Index(s.index).Id(s.id)
	if s.tenantName != "" {
		req = req.Header(SECURITY_TENANT_HEADER, s.tenantName)
	}
	result, err := req.Do(context.TODO())
	if elastic7.IsNotFound(err) {
		return nil, err // there is a check against this error
	}
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve dashboard object: %+v", err)
	}
	return result, nil
}

func elastic7DeleteDashboardObject(client *elastic7.Client, index, id, tenantName string) error {
	req := client.Delete().Index(index).Id(id)
	if tenantName != "" {
		req = req.Header(SECURITY_TENANT_HEADER, tenantName)
	}
	_, err := req.Do(context.TODO())

	// we'll get an error if it's not found
	return err
}
