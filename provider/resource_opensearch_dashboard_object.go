package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"

	elastic7 "github.com/olivere/elastic/v7"
)

func resourceOpensearchDashboardObject() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Dashboards object resource. This resource interacts directly with the underlying OpenSearch index backing Dashboards, so the format must match what Dashboards the version of Dashboards is expecting. Dashboards with older versions - directly pulling the JSON from a Dashboards index of the same version of OpenSearch targeted by the provider is a workaround.",
		Create:      resourceOpensearchDashboardObjectCreate,
		Read:        resourceOpensearchDashboardObjectRead,
		Update:      resourceOpensearchDashboardObjectUpdate,
		Delete:      resourceOpensearchDashboardObjectDelete,
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

						for _, k := range requiredDashboardObjectKeys() {
							if dashboardObject[k] == nil {
								errors = append(errors, fmt.Errorf("object must have the %q key", k))
							}
						}
					}

					return warnings, errors
				},
				// DiffSuppressFunc: diffSuppressDashboardObject,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				Description: "The JSON body of the dashboard object.",
			},
			"index": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     ".kibana",
				Description: "The name of the index where dashboard data is stored.",
			},
		},
	}
}

const (
	INDEX_CREATED int = iota
	INDEX_EXISTS
	INDEX_CREATION_FAILED
)

func resourceOpensearchDashboardObjectCreate(d *schema.ResourceData, meta interface{}) error {
	index := d.Get("index").(string)
	mapping_index := d.Get("index").(string)

	var success int
	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	success, err = elastic7CreateIndexIfNotExists(osClient, index, mapping_index)

	if err != nil {
		log.Printf("[INFO] Failed to create new Dashboard index: %+v", err)
		return err
	}

	if success == INDEX_CREATED {
		log.Printf("[INFO] Created new Dashboard index")
	} else if success == INDEX_CREATION_FAILED {
		return fmt.Errorf("fail to create OpenSearchsearch index")
	}

	id, err := resourceOpensearchPutDashboardObject(d, meta)

	if err != nil {
		log.Printf("[INFO] Failed to put Dashboard object: %+v", err)
		return err
	}

	d.SetId(id)
	log.Printf("[INFO] Object ID: %s", d.Id())

	return nil
}

func elastic7CreateIndexIfNotExists(client *elastic7.Client, index string, mappingIndex string) (int, error) {
	log.Printf("[INFO] elastic7CreateIndexIfNotExists %s", index)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(index).Do(context.TODO())
	if err != nil {
		return INDEX_CREATION_FAILED, err
	}
	if !exists {
		createIndex, err := client.CreateIndex(mappingIndex).Body(`{"mappings":{}}`).Do(context.TODO())
		if createIndex.Acknowledged {
			return INDEX_CREATED, err
		}
		return INDEX_CREATION_FAILED, err
	}

	return INDEX_EXISTS, nil
}

func resourceOpensearchDashboardObjectRead(d *schema.ResourceData, meta interface{}) error {
	bodyString := d.Get("body").(string)
	var body []interface{}
	if err := json.Unmarshal([]byte(bodyString), &body); err != nil {
		log.Printf("[WARN] Failed to unmarshal on read: %+v", bodyString)
		return err
	}
	dashboardObject, ok := body[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected %v to be an object", body[0])
	}
	id := dashboardObject["_id"].(string)
	index := d.Get("index").(string)

	var resultJSON []byte
	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	var result *elastic7.GetResult
	result, err = elastic7GetObject(osClient, index, id)
	if err == nil {
		resultJSON, err = json.Marshal(result)
	}

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] Dashboard Object (%s) not found, removing from state", id)
			d.SetId("")
			return nil
		}

		return err
	}
	log.Printf("[TRACE] body: %s", string(resultJSON))

	ds := &resourceDataSetter{d: d}
	ds.set("index", index)

	// The Dashboard object interface was originally built with the notion that
	// multiple Dashboard objects would be specified in the same resource, however,
	// that's not practical given that the OpenSearch API is for a single
	// object. We account for that here: use the _source attribute and build a
	// single entry array
	var originalKeys []string
	for k := range dashboardObject {
		originalKeys = append(originalKeys, k)
	}

	res := make(map[string]interface{})
	if err := json.Unmarshal(resultJSON, &res); err != nil {
		log.Printf("[WARN] Failed to unmarshal: %+v", resultJSON)
		return err
	}

	stateObject := []map[string]interface{}{make(map[string]interface{})}
	for _, k := range originalKeys {
		stateObject[0][k] = res[k]
	}
	state, err := json.Marshal(stateObject)
	if err != nil {
		return fmt.Errorf("error marshalling resource data: %+v", err)
	}
	ds.set("body", string(state))

	return ds.err
}

func resourceOpensearchDashboardObjectUpdate(d *schema.ResourceData, meta interface{}) error {
	_, err := resourceOpensearchPutDashboardObject(d, meta)
	return err
}

func resourceOpensearchDashboardObjectDelete(d *schema.ResourceData, meta interface{}) error {
	bodyString := d.Get("body").(string)
	var body []interface{}
	if err := json.Unmarshal([]byte(bodyString), &body); err != nil {
		log.Printf("[WARN] Failed to unmarshal: %+v", bodyString)
		return err
	}
	object, ok := body[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected %v to be an object", body[0])
	}
	id := object["_id"].(string)
	index := d.Get("index").(string)

	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	err = elastic7DeleteIndex(osClient, index, id)

	if err != nil {
		return err
	}

	return nil
}

func elastic7DeleteIndex(client *elastic7.Client, index string, id string) error {
	_, err := client.Delete().
		Index(index).
		Id(id).
		Do(context.TODO())

	// we'll get an error if it's not found
	return err
}

func resourceOpensearchPutDashboardObject(d *schema.ResourceData, meta interface{}) (string, error) {
	bodyString := d.Get("body").(string)
	var body []interface{}
	if err := json.Unmarshal([]byte(bodyString), &body); err != nil {
		log.Printf("[WARN] Failed to unmarshal on put: %+v", bodyString)
		return "", err
	}
	object, ok := body[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("expected %v to be an object", body[0])
	}
	id := object["_id"].(string)
	data := object["_source"]
	index := d.Get("index").(string)

	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return "", err
	}
	err = elastic7PutIndex(osClient, index, id, data)

	if err != nil {
		return "", err
	}

	return id, nil
}

func elastic7PutIndex(client *elastic7.Client, index string, id string, data interface{}) error {
	_, err := client.Index().
		Index(index).
		Id(id).
		BodyJson(&data).
		Do(context.TODO())

	return err
}

func requiredDashboardObjectKeys() []string {
	return []string{"_source", "_id"}
}
