package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var openDistroMonitorSchema = map[string]*schema.Schema{
	"body": {
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: diffSuppressMonitor,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
		ValidateFunc: validation.StringIsJSON,
	},
}

func resourceOpenSearchMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchOpenDistroMonitorCreate,
		Read:   resourceOpensearchOpenDistroMonitorRead,
		Update: resourceOpensearchOpenDistroMonitorUpdate,
		Delete: resourceOpensearchOpenDistroMonitorDelete,
		Schema: openDistroMonitorSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchOpenDistroMonitorCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchOpenDistroPostMonitor(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to put monitor: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Object ID: %s", d.Id())

	// Although we receive the full monitor in the response to the POST,
	// OpenDistro seems to add default values to the ojbect after the resource
	// is saved, e.g. adjust_pure_negative, boost values
	return resourceOpensearchOpenDistroMonitorRead(d, m)
}

func resourceOpensearchOpenDistroMonitorRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchOpenDistroGetMonitor(d.Id(), m)

	if elastic7.IsNotFound(err) {
		log.Printf("[WARN] Monitor (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	d.SetId(res.ID)

	monitorJson, err := json.Marshal(res.Monitor)
	if err != nil {
		return err
	}
	monitorJsonNormalized, err := structure.NormalizeJsonString(string(monitorJson))
	if err != nil {
		return err
	}
	err = d.Set("body", monitorJsonNormalized)
	return err
}

func resourceOpensearchOpenDistroMonitorUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchOpenDistroPutMonitor(d, m)

	if err != nil {
		return err
	}

	return resourceOpensearchOpenDistroMonitorRead(d, m)
}

func resourceOpensearchOpenDistroMonitorDelete(d *schema.ResourceData, m interface{}) error {
	var err error

	path, err := uritemplates.Expand("/_opendistro/_alerting/monitors/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for monitor: %+v", err)
	}

	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "DELETE",
		Path:   path,
	})

	return err
}

func resourceOpensearchOpenDistroGetMonitor(monitorID string, m interface{}) (*monitorResponse, error) {
	var err error
	response := new(monitorResponse)

	path, err := uritemplates.Expand("/_opendistro/_alerting/monitors/{id}", map[string]string{
		"id": monitorID,
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for monitor: %+v", err)
	}

	var body json.RawMessage
	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling monitor body: %+v: %+v", err, body)
	}
	normalizeMonitor(response.Monitor)
	return response, err
}

func resourceOpensearchOpenDistroPostMonitor(d *schema.ResourceData, m interface{}) (*monitorResponse, error) {
	monitorJSON := d.Get("body").(string)

	var err error
	response := new(monitorResponse)

	path := "/_opendistro/_alerting/monitors/"

	var body json.RawMessage
	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   monitorJSON,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling monitor body: %+v: %+v", err, body)
	}
	normalizeMonitor(response.Monitor)
	return response, nil
}

func resourceOpensearchOpenDistroPutMonitor(d *schema.ResourceData, m interface{}) (*monitorResponse, error) {
	monitorJSON := d.Get("body").(string)

	var err error
	response := new(monitorResponse)

	path, err := uritemplates.Expand("/_opendistro/_alerting/monitors/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for monitor: %+v", err)
	}

	var body json.RawMessage
	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "PUT",
		Path:   path,
		Body:   monitorJSON,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling monitor body: %+v: %+v", err, body)
	}

	return response, nil
}

type monitorResponse struct {
	Version int                    `json:"_version"`
	ID      string                 `json:"_id"`
	Monitor map[string]interface{} `json:"monitor"`
}
