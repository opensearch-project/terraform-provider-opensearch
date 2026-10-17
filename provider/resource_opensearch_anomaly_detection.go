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
	"enabled": {
		Description: "Whether the real-time job of the anomaly detector runs. When set, the provider starts or stops the job after create and update. When not set, the job state is left as it is (an update stops a running job for the PUT and starts it again). The detector is always stopped before delete.",
		Type:        schema.TypeBool,
		Optional:    true,
		Computed:    true,
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

	if enabled, ok := anomalyDetectionEnabledInConfig(d); ok {
		if err := resourceOpensearchAnomalyDetectionSetEnabled(d.Id(), enabled, m); err != nil {
			return err
		}
	}

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
	if err := d.Set("body", anomalyDetectionJsonNormalized); err != nil {
		return err
	}

	if err := d.Set("enabled", res.JobEnabled); err != nil {
		return err
	}

	return nil
}

func resourceOpensearchAnomalyDetectionUpdate(d *schema.ResourceData, m interface{}) error {
	// OpenSearch rejects a PUT on a running detector ("Job is running"). Stop it
	// first so the body update always succeeds, then start it again when the
	// config asks for it or, with enabled unset, when it was running before.
	cur, err := resourceOpensearchAnomalyDetectionGet(d.Id(), m)
	if err != nil {
		return err
	}
	desired := cur.JobEnabled
	if enabled, ok := anomalyDetectionEnabledInConfig(d); ok {
		desired = enabled
	}

	if d.HasChange("body") {
		if err := resourceOpensearchAnomalyDetectionSetEnabled(d.Id(), false, m); err != nil {
			return err
		}
		if _, err := resourceOpensearchPutAnomalyDetection(d, m); err != nil {
			return err
		}
	}

	if err := resourceOpensearchAnomalyDetectionSetEnabled(d.Id(), desired, m); err != nil {
		return err
	}

	return resourceOpensearchAnomalyDetectionRead(d, m)
}

func resourceOpensearchAnomalyDetectionGet(anomalyDetectionID string, m interface{}) (*anomalyDetectionResponse, error) {
	var err error
	response := new(anomalyDetectionResponse)

	// job=true adds anomaly_detector_job, the only place the real-time job state
	// is returned; anomaly_detector itself never carries a detector_state.
	path, err := uritemplates.Expand("/_plugins/_anomaly_detection/detectors/{id}?job=true", map[string]string{
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
	if response.Job != nil {
		response.JobEnabled, _ = response.Job["enabled"].(bool)
	}
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
	if err := resourceOpensearchAnomalyDetectionSetEnabled(d.Id(), false, m); err != nil {
		log.Printf("[WARN] Failed to stop anomaly detector %s before delete: %+v", d.Id(), err)
		// proceed with delete attempt anyway
	}

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

// resourceOpensearchAnomalyDetectionSetEnabled starts or stops the detector
// when the desired state differs from the current live state.
func resourceOpensearchAnomalyDetectionSetEnabled(id string, desired bool, m interface{}) error {
	res, err := resourceOpensearchAnomalyDetectionGet(id, m)
	if err != nil {
		return err
	}
	if desired == res.JobEnabled {
		return nil // already in desired state
	}

	action := "_stop"
	if desired {
		action = "_start"
	}

	path, err := uritemplates.Expand("/_plugins/_anomaly_detection/detectors/{id}/"+action, map[string]string{"id": id})
	if err != nil {
		return fmt.Errorf("error building URL for anomaly detector %s: %+v", action, err)
	}
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
	})
	return err
}

type anomalyDetectionResponse struct {
	Version         int                    `json:"_version"`
	ID              string                 `json:"_id"`
	AnomalyDetector map[string]interface{} `json:"anomaly_detector"`
	Job             map[string]interface{} `json:"anomaly_detector_job"`
	// JobEnabled is true while the real-time job runs (anomaly_detector_job.enabled).
	JobEnabled bool
}

// anomalyDetectionEnabledInConfig returns the configured enabled value and
// whether it is set at all (it is Computed, so d.Get cannot tell unset from false).
func anomalyDetectionEnabledInConfig(d *schema.ResourceData) (bool, bool) {
	v := d.GetRawConfig().GetAttr("enabled")
	if v.IsNull() || !v.IsKnown() {
		return false, false
	}
	return v.True(), true
}
