package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
		Description: "Provides an OpenSearch Dashboards object resource. This resource interacts directly with the underlying OpenSearch index backing Dashboards, so the format must match what Dashboards the version of Dashboards is expecting. Dashboards with older versions - directly pulling the JSON from a Dashboards index of the same version of OpenSearch targeted by the provider is a workaround. For index patterns, per-field popularity counters (`fields[].count`) are ignored when diffing: Dashboards increments them as people use fields in Discover, so they are usage telemetry rather than configuration.",
		Create:      resourceOpensearchDashboardObjectCreate,
		Read:        resourceOpensearchDashboardObjectRead,
		Update:      resourceOpensearchDashboardObjectUpdate,
		Delete:      resourceOpensearchDashboardObjectDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceOpensearchDashboardObjectImport,
		},
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
				return dashboardObjectOld["_id"] != dashboardObjectNew["_id"]
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
				DiffSuppressFunc: dashboardObjectPopularityDiffSuppress,
				Description:      "The JSON body of the dashboard object.",
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

// Dashboards increments an index pattern's per-field popularity counter every time someone
// uses that field in Discover, writing it back into the saved object. It is usage telemetry
// rather than configuration, so a difference confined to those counters is not drift worth
// reporting: reverting them would only churn the document and they would climb again.
func dashboardObjectPopularityDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	oldStripped, err := stripDashboardFieldPopularity(old)
	if err != nil {
		return false
	}
	newStripped, err := stripDashboardFieldPopularity(new)
	if err != nil {
		return false
	}
	return oldStripped == newStripped
}

// stripDashboardFieldPopularity returns body with every index pattern's `count` field removed,
// canonicalised so two bodies differing only in those counters compare equal. Anything that
// does not parse as the expected shape is left untouched rather than dropped, so an unexpected
// document still diffs on its real contents.
func stripDashboardFieldPopularity(body string) (string, error) {
	var objects []interface{}
	if err := json.Unmarshal([]byte(body), &objects); err != nil {
		return "", err
	}

	for _, o := range objects {
		object, ok := o.(map[string]interface{})
		if !ok {
			continue
		}
		source, ok := object["_source"].(map[string]interface{})
		if !ok {
			continue
		}
		objectType, ok := source["type"].(string)
		if !ok || objectType != "index-pattern" {
			continue
		}
		attributes, ok := source[objectType].(map[string]interface{})
		if !ok {
			continue
		}
		// Attributes hold `fields` as a JSON-encoded string, not a nested array.
		encodedFields, ok := attributes["fields"].(string)
		if !ok {
			continue
		}
		var fields []interface{}
		if err := json.Unmarshal([]byte(encodedFields), &fields); err != nil {
			continue
		}

		stripped := false
		for _, f := range fields {
			field, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			if _, ok := field["count"]; ok {
				delete(field, "count")
				stripped = true
			}
		}
		if !stripped {
			continue
		}

		reencoded, err := json.Marshal(fields)
		if err != nil {
			continue
		}
		attributes["fields"] = string(reencoded)
	}

	// json.Marshal sorts object keys, so the result is canonical.
	canonical, err := json.Marshal(objects)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

func resourceOpensearchDashboardObjectImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	objectID, tenantName, indexName, err := parseDashboardObjectImportID(d.Id())
	if err != nil {
		return nil, err
	}

	if tenantName != "" {
		if err := d.Set("tenant_name", tenantName); err != nil {
			return nil, err
		}
	}
	if indexName != "" {
		if err := d.Set("index", indexName); err != nil {
			return nil, err
		}
	}

	d.SetId(objectID)

	seedBodyBytes, err := json.Marshal([]map[string]interface{}{{
		"_id":     objectID,
		"_source": map[string]interface{}{},
	}})
	if err != nil {
		return nil, err
	}

	if err := d.Set("body", string(seedBodyBytes)); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func resourceOpensearchDashboardObjectCreate(d *schema.ResourceData, meta interface{}) error {
	// parse desired terrafrom state
	state, err := readDashboardObjectState(d)
	if err != nil {
		return err
	}
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return fmt.Errorf("could not read client: %+v", err)
	}

	// make OpenSearch API calls
	if err = elastic7CreateIndexIfNotExists(client, state.index); err != nil {
		return fmt.Errorf("failed to create new Dashboard index: %+v", err)
	}
	resp, err := state.elastic7PutDashboardObject(client)
	if err != nil {
		return fmt.Errorf("failed to put Dashboard object: %+v", err)
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
		return fmt.Errorf("could not read state from OpenSearch: %+v", err)
	}

	// build json string from response that represents body configuration
	// Note that only the 'original' keys are considered. Keys that
	// OpenSearch adds internally will be ignored (e.g. 'updated_at').
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %+v", err)
	}
	log.Printf("[TRACE] body: %s", string(resultJSON))

	var originalKeys []string
	for k := range state.dashboardObject {
		originalKeys = append(originalKeys, k)
	}

	res := make(map[string]interface{})
	if err := json.Unmarshal(resultJSON, &res); err != nil {
		return fmt.Errorf("failed to unmarshal '%+v': %+v", resultJSON, err)
	}

	stateObject := []map[string]interface{}{make(map[string]interface{})}
	for _, k := range originalKeys {
		stateObject[0][k] = res[k]
	}
	bodyBytes, err := json.Marshal(stateObject)
	if err != nil {
		return fmt.Errorf("failed marshalling resource data: %+v", err)
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
		return fmt.Errorf("dashboard object update failed: %+v", err)
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
	return elastic7DeleteDashboardObject(client, indexStr, d.Id(), tenantNameStr)
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
		return fmt.Errorf("failed to create OpenSearch search index: %+v", err)
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
		if d.Id() == "" {
			return nil, fmt.Errorf("could not read body interface: %+v", err)
		}
		dashboardObject = map[string]interface{}{
			"_id":     d.Id(),
			"_source": map[string]interface{}{},
		}
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

	objectID, ok := dashboardObject["_id"].(string)
	if !ok || objectID == "" {
		if d.Id() == "" {
			return nil, fmt.Errorf("dashboard object body missing valid _id")
		}
		objectID = d.Id()
		dashboardObject["_id"] = objectID
	}

	if dashboardObject["_source"] == nil {
		dashboardObject["_source"] = map[string]interface{}{}
	}

	return &dashboardObjectState{
		index:           indexName,
		tenantName:      tenantName,
		dashboardObject: dashboardObject,
		id:              objectID,
	}, nil
}

func parseDashboardObjectImportID(input string) (string, string, string, error) {
	if input == "" {
		return "", "", "", fmt.Errorf("invalid import ID: expected object_id[,tenant_name=<name>][,index=<name>]")
	}

	parts := strings.Split(input, ",")
	objectID := strings.TrimSpace(parts[0])
	if objectID == "" {
		return "", "", "", fmt.Errorf("invalid import ID %q: object_id cannot be empty", input)
	}

	tenantName := ""
	indexName := ""

	for _, raw := range parts[1:] {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return "", "", "", fmt.Errorf("invalid import ID %q: expected key=value segment %q", input, part)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if value == "" {
			return "", "", "", fmt.Errorf("invalid import ID %q: value for %q cannot be empty", input, key)
		}

		switch key {
		case "tenant_name":
			if tenantName != "" {
				return "", "", "", fmt.Errorf("invalid import ID %q: tenant_name provided more than once", input)
			}
			tenantName = value
		case "index":
			if indexName != "" {
				return "", "", "", fmt.Errorf("invalid import ID %q: index provided more than once", input)
			}
			indexName = value
		default:
			return "", "", "", fmt.Errorf("invalid import ID %q: unsupported key %q", input, key)
		}
	}

	if tenantName != "" && indexName != "" {
		return "", "", "", fmt.Errorf("invalid import ID %q: tenant_name conflicts with index", input)
	}

	return objectID, tenantName, indexName, nil
}

func readBodyInterface(i interface{}) (map[string]interface{}, error) {
	bodyString, ok := i.(string)
	if !ok {
		return nil, fmt.Errorf("cannot convert input to string")
	}

	var body []interface{}
	if err := json.Unmarshal([]byte(bodyString), &body); err != nil {
		return nil, fmt.Errorf("could not unmarshal body string: %+v", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("body has no elements as JSON array")
	}

	dashboardObject, ok := body[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("body has unexpected format")
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
		return nil, fmt.Errorf("could not retrieve dashboard object: %+v", err)
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
