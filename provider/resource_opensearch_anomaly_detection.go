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

	if elastic7.IsNotFound(err) {
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
	var err error
	response := new(anomalyDetectionResponse)

	path, err := uritemplates.Expand("/_plugins/_anomaly_detection/detectors/{id}", map[string]string{
		"id": anomalyDetectionID,
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for anomaly detector: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling anomaly detector body: %+v: %+v", err, body)
	}
	log.Printf("[INFO] Response: %+v", response)
	normalizeAnomalyDetection(response.AnomalyDetector)
	log.Printf("[INFO] Response: %+v", response)
	log.Printf("The version %v", response.Version)
	return response, err
}

func resourceOpensearchPostAnomalyDetection(d *schema.ResourceData, m interface{}) (*anomalyDetectionResponse, error) {
	anomalyDetectionJSON := d.Get("body").(string)

	var err error
	response := new(anomalyDetectionResponse)

	path := "/_plugins/_anomaly_detection/detectors/"

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   anomalyDetectionJSON,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling anomaly detector body: %+v: %+v", err, body)
	}
	normalizeAnomalyDetection(response.AnomalyDetector)
	return response, nil
}

func resourceOpensearchPutAnomalyDetection(d *schema.ResourceData, m interface{}) (*anomalyDetectionResponse, error) {
	anomalyDetectionJSON := d.Get("body").(string)

	var err error
	response := new(anomalyDetectionResponse)

	path, err := uritemplates.Expand("/_plugins/_anomaly_detection/detectors/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for anomaly detector: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "PUT",
		Path:   path,
		Body:   anomalyDetectionJSON,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling anomaly detector body: %+v: %+v", err, body)
	}

	return response, nil
}

func resourceOpensearchAnomalyDetectionDelete(d *schema.ResourceData, m interface{}) error {
	var err error

	path, err := uritemplates.Expand("/_plugins/_anomaly_detection/detectors/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for anomaly detector: %+v", err)
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "DELETE",
		Path:   path,
	})

	return err
}

type anomalyDetectionResponse struct {
	Version         int                    `json:"_version"`
	ID              string                 `json:"_id"`
	AnomalyDetector map[string]interface{} `json:"anomaly_detector"`
}
