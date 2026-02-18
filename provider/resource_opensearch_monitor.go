package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var openDistroMonitorSchema = map[string]*schema.Schema{
	"body": {
		Description:      "The monitor document",
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
		Description: "Provides an OpenSearch monitor. Please refer to the OpenSearch monitor documentation for details.",
		Create:      resourceOpensearchOpenDistroMonitorCreate,
		Read:        resourceOpensearchOpenDistroMonitorRead,
		Update:      resourceOpensearchOpenDistroMonitorUpdate,
		Delete:      resourceOpensearchOpenDistroMonitorDelete,
		Schema:      openDistroMonitorSchema,
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

	if isNotFound(err) {
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

	path := fmt.Sprintf("/_plugins/_alerting/monitors/%s", d.Id())

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	// Build request
	req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	if err != nil {
		return fmt.Errorf("error building DELETE request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return fmt.Errorf("error deleting monitor: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting monitor: received status code %d", resp.StatusCode)
}

func resourceOpensearchOpenDistroGetMonitor(monitorID string, m interface{}) (*monitorResponse, error) {
	var err error
	response := new(monitorResponse)

	path := fmt.Sprintf("/_plugins/_alerting/monitors/%s", monitorID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return response, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error getting monitor: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return response, fmt.Errorf("monitor not found: %s", monitorID)
	}

	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("error getting monitor: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling monitor body: %+v: %+v", err, body)
	}
	log.Printf("[INFO] Response: %+v", response)
	normalizeMonitor(response.Monitor)
	log.Printf("[INFO] Response: %+v", response)
	log.Printf("The version %v", response.Version)
	return response, err
}

func resourceOpensearchOpenDistroPostMonitor(d *schema.ResourceData, m interface{}) (*monitorResponse, error) {
	monitorJSON := d.Get("body").(string)

	response := new(monitorResponse)
	path := "/_plugins/_alerting/monitors/"

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("POST", client.config.rawUrl+path, strings.NewReader(monitorJSON))
	if err != nil {
		return response, fmt.Errorf("error building POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error posting monitor: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error posting monitor: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling monitor body: %+v: %+v", err, body)
	}
	normalizeMonitor(response.Monitor)
	return response, nil
}

func resourceOpensearchOpenDistroPutMonitor(d *schema.ResourceData, m interface{}) (*monitorResponse, error) {
	monitorJSON := d.Get("body").(string)

	response := new(monitorResponse)
	path := fmt.Sprintf("/_plugins/_alerting/monitors/%s", d.Id())

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(monitorJSON))
	if err != nil {
		return response, fmt.Errorf("error building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error putting monitor: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error putting monitor: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling monitor body: %+v: %+v", err, body)
	}

	return response, nil
}

type monitorResponse struct {
	Version int                    `json:"_version"`
	ID      string                 `json:"_id"`
	Name    string                 `json:"name"`
	Monitor map[string]interface{} `json:"monitor"`
}
