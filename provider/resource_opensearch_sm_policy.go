package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var openSearchSMPolicySchema = map[string]*schema.Schema{
	"policy_name": {
		Description: "The name of the SM policy.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"body": {
		Description:      "The policy document.",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: smDiffSuppressPolicy,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
	},
	"primary_term": {
		Description: "The primary term of the SM policy version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"seq_no": {
		Description: "The sequence number of the SM policy version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
}

func resourceOpenSearchSMPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Snapshot Management (SM) policy. Please refer to the OpenSearch SM documentation for details.",
		Create:      resourceOpensearchSMPolicyCreate,
		Read:        resourceOpensearchSMPolicyRead,
		Update:      resourceOpensearchSMPolicyUpdate,
		Delete:      resourceOpensearchSMPolicyDelete,
		Schema:      openSearchSMPolicySchema,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				var err error = d.Set("policy_name", d.Id())
				if err != nil {
					return nil, err
				}

				d.SetId(fmt.Sprintf("%s-sm-policy", d.Id()))
				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func resourceOpensearchSMPolicyCreate(d *schema.ResourceData, m interface{}) error {
	policyResponse, err := resourceOpensearchPostPutSMPolicy(d, m, "POST")

	if err != nil {
		log.Printf("[INFO] Failed to create OpenSearchPolicy: %+v", err)
		return err
	}

	d.SetId(policyResponse.PolicyID)
	return resourceOpensearchSMPolicyRead(d, m)
}

func resourceOpensearchSMPolicyRead(d *schema.ResourceData, m interface{}) error {
	policyResponse, err := resourceOpensearchGetSMPolicy(d.Get("policy_name").(string), m)

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] OpenSearch Policy (%s) not found, removing from state", d.Get("policy_name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	bodyString, err := json.Marshal(policyResponse.Policy)
	if err != nil {
		return err
	}

	bodyStringNormalized, _ := structure.NormalizeJsonString(string(bodyString))

	if err := d.Set("policy_name", policyResponse.Policy["name"]); err != nil {
		return fmt.Errorf("error setting policy_name: %s", err)
	}
	if err := d.Set("body", bodyStringNormalized); err != nil {
		return fmt.Errorf("error setting body: %s", err)
	}
	if err := d.Set("primary_term", policyResponse.PrimaryTerm); err != nil {
		return fmt.Errorf("error setting primary_term: %s", err)
	}
	if err := d.Set("seq_no", policyResponse.SeqNo); err != nil {
		return fmt.Errorf("error setting seq_no: %s", err)
	}

	return nil
}

func resourceOpensearchSMPolicyUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPostPutSMPolicy(d, m, "PUT"); err != nil {
		return err
	}

	return resourceOpensearchSMPolicyRead(d, m)
}

func resourceOpensearchSMPolicyDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_plugins/_sm/policies/{policy_name}", map[string]string{
		"policy_name": d.Get("policy_name").(string),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for policy: %+v", err)
	}

	osclient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osclient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           "DELETE",
		Path:             path,
		RetryStatusCodes: []int{http.StatusConflict},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})

	if err != nil {
		return fmt.Errorf("error deleting policy: %+v : %+v", path, err)
	}

	return err
}

func resourceOpensearchGetSMPolicy(policyName string, m interface{}) (SMPolicyResponse, error) {
	var err error
	response := new(SMPolicyResponse)

	path, err := uritemplates.Expand("/_plugins/_sm/policies/{policy_name}", map[string]string{
		"policy_name": policyName,
	})

	if err != nil {
		return *response, fmt.Errorf("error building URL path for policy: %+v", err)
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
		return *response, fmt.Errorf("error getting policy: %+v : %+v", path, err)
	}
	body = &res.Body

	if err != nil {
		return *response, err
	}

	if err := json.Unmarshal(*body, &response); err != nil {
		return *response, fmt.Errorf("error unmarshalling policy body: %+v: %+v", err, body)
	}

	normalizePolicy(response.Policy)

	return *response, err
}

func resourceOpensearchPostPutSMPolicy(d *schema.ResourceData, m interface{}, method string) (*SMPolicyResponse, error) {
	response := new(SMPolicyResponse)
	policyJSON := d.Get("body").(string)
	seq := d.Get("seq_no").(int)
	primTerm := d.Get("primary_term").(int)
	params := url.Values{}

	if seq >= 0 && primTerm > 0 {
		params.Set("if_seq_no", strconv.Itoa(seq))
		params.Set("if_primary_term", strconv.Itoa(primTerm))
	}

	path, err := uritemplates.Expand("/_plugins/_sm/policies/{policy_name}", map[string]string{
		"policy_name": d.Get("policy_name").(string),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for policy: %+v", err)
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
		Body:             string(policyJSON),
		RetryStatusCodes: []int{http.StatusConflict},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})
	if err != nil {
		return response, fmt.Errorf("error posting policy: %+v : %+v : %+v", path, policyJSON, err)
	}
	body = &res.Body

	if err != nil {
		return response, fmt.Errorf("error creating policy mapping: %+v", err)
	}

	if err := json.Unmarshal(*body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling policy body: %+v: %+v", err, body)
	}

	return response, nil
}

type SMPolicyResponse struct {
	PolicyID    string                 `json:"_id"`
	Version     int                    `json:"_version"`
	PrimaryTerm int                    `json:"_primary_term"`
	SeqNo       int                    `json:"_seq_no"`
	Policy      map[string]interface{} `json:"sm_policy"`
}

func smDiffSuppressPolicy(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	om, ok := oo.(map[string]interface{})
	if ok {
		normalizePolicy(om)
	}

	nm, ok := no.(map[string]interface{})
	if ok {
		normalizePolicy(nm)
	}

	// Suppress diff of autogenerated fields by copying them to the old object
	if name, ok := om["name"]; ok {
		nm["name"] = name
	}

	if enabled_time, ok := om["enabled_time"]; ok {
		nm["enabled_time"] = enabled_time
	}

	if schedule, ok := om["schedule"]; ok {
		nm["schedule"] = schedule
	}

	return reflect.DeepEqual(oo, no)
}
