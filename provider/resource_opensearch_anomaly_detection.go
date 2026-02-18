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

var anomalyDetectionSchema = map[string]*schema.Schema{
	"body": {
		Description:      "The anomaly detection document",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: diffSuppressAnomalyDetection,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
		ValidateFunc: validation.StringIsJSON,
	},
}

func resourceOpenSearchAnomalyDetection() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch anonaly detection. Please refer to the OpenSearch anomaly detection documentation for details.",
		Create:      resourceOpensearchAnomalyDetectionCreate,
		Read:        resourceOpensearchAnomalyDetectionRead,
		Update:      resourceOpensearchAnomalyDetectionUpdate,
		Delete:      resourceOpensearchAnomalyDetectionDelete,
		Schema:      anomalyDetectionSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchAnomalyDetectionCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchPostAnomalyDetection(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to put anomaly detector: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Object ID: %s", d.Id())

	return resourceOpensearchAnomalyDetectionRead(d, m)
}

func resourceOpensearchAnomalyDetectionRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchAnomalyDetectionGet(d.Id(), m)

	if isNotFound(err) {
		log.Printf("[WARN] Anomaly Detector (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	d.SetId(res.ID)

	anomalyDetectionJSON, err := json.Marshal(res.AnomalyDetector)
	if err != nil {
		return err
	}
	anomalyDetectionJsonNormalized, err := structure.NormalizeJsonString(string(anomalyDetectionJSON))
	if err != nil {
		return err
	}
	err = d.Set("body", anomalyDetectionJsonNormalized)
	return err
}

func resourceOpensearchAnomalyDetectionUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchPutAnomalyDetection(d, m)

	if err != nil {
		return err
	}

	return resourceOpensearchAnomalyDetectionRead(d, m)
}

func resourceOpensearchAnomalyDetectionGet(anomalyDetectionID string, m interface{}) (*anomalyDetectionResponse, error) {
	response := new(anomalyDetectionResponse)

	path := fmt.Sprintf("/_plugins/_anomaly_detection/detectors/%s", anomalyDetectionID)

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
		return response, fmt.Errorf("error getting anomaly detector: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return response, fmt.Errorf("anomaly detector not found: %s", anomalyDetectionID)
	}

	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("error getting anomaly detector: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling anomaly detector body: %+v: %+v", err, body)
	}
	log.Printf("[INFO] Response: %+v", response)
	normalizeAnomalyDetection(response.AnomalyDetector)
	log.Printf("[INFO] Response: %+v", response)
	log.Printf("The version %v", response.Version)
	return response, nil
}

func resourceOpensearchPostAnomalyDetection(d *schema.ResourceData, m interface{}) (*anomalyDetectionResponse, error) {
	anomalyDetectionJSON := d.Get("body").(string)

	response := new(anomalyDetectionResponse)
	path := "/_plugins/_anomaly_detection/detectors/"

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("POST", client.config.rawUrl+path, strings.NewReader(anomalyDetectionJSON))
	if err != nil {
		return response, fmt.Errorf("error building POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error posting anomaly detector: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error posting anomaly detector: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling anomaly detector body: %+v: %+v", err, body)
	}
	normalizeAnomalyDetection(response.AnomalyDetector)
	return response, nil
}

func resourceOpensearchPutAnomalyDetection(d *schema.ResourceData, m interface{}) (*anomalyDetectionResponse, error) {
	anomalyDetectionJSON := d.Get("body").(string)

	response := new(anomalyDetectionResponse)
	path := fmt.Sprintf("/_plugins/_anomaly_detection/detectors/%s", d.Id())

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(anomalyDetectionJSON))
	if err != nil {
		return response, fmt.Errorf("error building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error putting anomaly detector: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error putting anomaly detector: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling anomaly detector body: %+v: %+v", err, body)
	}

	return response, nil
}

func resourceOpensearchAnomalyDetectionDelete(d *schema.ResourceData, m interface{}) error {
	path := fmt.Sprintf("/_plugins/_anomaly_detection/detectors/%s", d.Id())

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
		return fmt.Errorf("error deleting anomaly detector: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting anomaly detector: received status code %d", resp.StatusCode)
}

type anomalyDetectionResponse struct {
	Version         int                    `json:"_version"`
	ID              string                 `json:"_id"`
	AnomalyDetector map[string]interface{} `json:"anomaly_detector"`
}
