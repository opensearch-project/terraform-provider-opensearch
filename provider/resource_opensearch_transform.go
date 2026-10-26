package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var openSearchTransformSchema = map[string]*schema.Schema{
	"transform_id": {
		Description: "The id of the transform job.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"body": {
		Description:      "The transform job document, wrapped in a top-level `transform` key.",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: transformDiffSuppressPolicy,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
	},
	"primary_term": {
		Description: "The primary term of the transform job version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"seq_no": {
		Description: "The sequence number of the transform job version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
}

func resourceOpensearchTransform() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Transform job. Please refer to the OpenSearch Transforms documentation for details.",
		Create:      resourceOpensearchTransformCreate,
		Read:        resourceOpensearchTransformRead,
		Update:      resourceOpensearchTransformUpdate,
		Delete:      resourceOpensearchTransformDelete,
		Schema:      openSearchTransformSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchTransformCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutTransform(d, m, "PUT"); err != nil {
		log.Printf("[INFO] Failed to create OpensearchTransform: %+v", err)
		return err
	}

	transformID := d.Get("transform_id").(string)
	d.SetId(transformID)

	if err := resourceOpensearchSetTransformEnabled(d, m); err != nil {
		return err
	}

	return resourceOpensearchTransformRead(d, m)
}

func resourceOpensearchTransformRead(d *schema.ResourceData, m interface{}) error {
	transformResponse, err := resourceOpensearchGetTransform(d.Id(), m)

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] OpenSearch Transform (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	bodyString, err := json.Marshal(transformResponse.Transform)
	if err != nil {
		return err
	}

	// Need encapsulation as the response from the GET is different than the one in the PUT
	bodyStringNormalized, _ := structure.NormalizeJsonString(fmt.Sprintf("{\"transform\": %+s}", string(bodyString)))

	if err := d.Set("transform_id", transformResponse.ID); err != nil {
		return fmt.Errorf("error setting transform_id: %s", err)
	}
	if err := d.Set("body", bodyStringNormalized); err != nil {
		return fmt.Errorf("error setting body: %s", err)
	}
	if err := d.Set("primary_term", transformResponse.PrimaryTerm); err != nil {
		return fmt.Errorf("error setting primary_term: %s", err)
	}
	if err := d.Set("seq_no", transformResponse.SeqNo); err != nil {
		return fmt.Errorf("error setting seq_no: %s", err)
	}

	return nil
}

func resourceOpensearchTransformUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutTransform(d, m, "PUT"); err != nil {
		return err
	}

	if err := resourceOpensearchSetTransformEnabled(d, m); err != nil {
		return err
	}

	return resourceOpensearchTransformRead(d, m)
}

func resourceOpensearchTransformDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_plugins/_transform/{transform_id}", map[string]string{
		"transform_id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for transform: %+v", err)
	}

	params := url.Values{}
	params.Set("force", "true")

	osclient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osclient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           "DELETE",
		Path:             path,
		Params:           params,
		RetryStatusCodes: []int{http.StatusConflict},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})

	if err != nil {
		return fmt.Errorf("error deleting transform: %+v : %+v", path, err)
	}

	return err
}

func resourceOpensearchGetTransform(transformID string, m interface{}) (TransformResponse, error) {
	var err error
	response := new(TransformResponse)

	path, err := uritemplates.Expand("/_plugins/_transform/{transform_id}", map[string]string{
		"transform_id": transformID,
	})

	if err != nil {
		return *response, fmt.Errorf("error building URL path for transform: %+v", err)
	}

	var body *json.RawMessage
	osclient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return *response, err
	}
	var res *elastic7.Response
	res, err = osclient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})

	if err != nil {
		return *response, fmt.Errorf("error getting transform: %+v : %+v", path, err)
	}
	body = &res.Body

	if err := json.Unmarshal(*body, &response); err != nil {
		return *response, fmt.Errorf("error unmarshalling transform body: %+v: %+v", err, body)
	}

	normalizeTransform(response.Transform)

	return *response, err
}

func resourceOpensearchPutTransform(d *schema.ResourceData, m interface{}, method string) (*TransformResponse, error) {
	response := new(TransformResponse)
	transformJSON := d.Get("body").(string)
	seq := d.Get("seq_no").(int)
	primTerm := d.Get("primary_term").(int)
	params := url.Values{}

	if seq >= 0 && primTerm > 0 {
		params.Set("if_seq_no", strconv.Itoa(seq))
		params.Set("if_primary_term", strconv.Itoa(primTerm))
	}

	path, err := uritemplates.Expand("/_plugins/_transform/{transform_id}", map[string]string{
		"transform_id": d.Get("transform_id").(string),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for transform: %+v", err)
	}

	var body *json.RawMessage
	osclient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osclient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           method,
		Path:             path,
		Params:           params,
		Body:             string(transformJSON),
		RetryStatusCodes: []int{http.StatusConflict},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})
	if err != nil {
		return response, fmt.Errorf("error putting transform: %+v : %+v : %+v", path, transformJSON, err)
	}
	body = &res.Body

	if err := json.Unmarshal(*body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling transform body: %+v: %+v", err, body)
	}

	return response, nil
}

// resourceOpensearchSetTransformEnabled starts or stops the transform job so that its
// actual execution state matches the "enabled" field in the configured body. Creating
// or updating a transform job's config via PUT does not by itself start execution.
func resourceOpensearchSetTransformEnabled(d *schema.ResourceData, m interface{}) error {
	transformID := d.Get("transform_id").(string)
	enabled := true

	var parsed struct {
		Transform struct {
			Enabled *bool `json:"enabled"`
		} `json:"transform"`
	}
	if err := json.Unmarshal([]byte(d.Get("body").(string)), &parsed); err != nil {
		return fmt.Errorf("error parsing transform body: %+v", err)
	}
	if parsed.Transform.Enabled != nil {
		enabled = *parsed.Transform.Enabled
	}

	action := "_stop"
	if enabled {
		action = "_start"
	}

	path, err := uritemplates.Expand("/_plugins/_transform/{transform_id}/"+action, map[string]string{
		"transform_id": transformID,
	})
	if err != nil {
		return fmt.Errorf("error building URL path for transform %s: %+v", action, err)
	}

	osclient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osclient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           "POST",
		Path:             path,
		RetryStatusCodes: []int{http.StatusConflict},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})
	if err != nil {
		return fmt.Errorf("error calling transform %s: %+v : %+v", action, path, err)
	}

	return nil
}

type TransformResponse struct {
	ID          string                 `json:"_id"`
	Version     int                    `json:"_version"`
	PrimaryTerm int                    `json:"_primary_term"`
	SeqNo       int                    `json:"_seq_no"`
	Transform   map[string]interface{} `json:"transform"`
}
