package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
)

var openSearchISMPolicySchema = map[string]*schema.Schema{
	"policy_id": {
		Description: "The id of the ISM policy.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"body": {
		Description:      "The policy document.",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: diffSuppressPolicy,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
	},
	"primary_term": {
		Description: "The primary term of the ISM policy version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"seq_no": {
		Description: "The sequence number of the ISM policy version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
}

func resourceOpenSearchISMPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Index State Management (ISM) policy. Please refer to the OpenSearch ISM documentation for details.",
		Create:      resourceOpensearchISMPolicyCreate,
		Read:        resourceOpensearchISMPolicyRead,
		Update:      resourceOpensearchISMPolicyUpdate,
		Delete:      resourceOpensearchISMPolicyDelete,
		Schema:      openSearchISMPolicySchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchISMPolicyCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutISMPolicy(d, m); err != nil {
		log.Printf("[INFO] Failed to create OpensearchPolicy: %+v", err)
		return err
	}

	policyID := d.Get("policy_id").(string)
	d.SetId(policyID)
	return resourceOpensearchISMPolicyRead(d, m)
}

func resourceOpensearchISMPolicyRead(d *schema.ResourceData, m interface{}) error {
	policyResponse, err := resourceOpensearchGetISMPolicy(d.Id(), m)

	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] OpenSearch Policy (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	bodyString, err := json.Marshal(policyResponse.Policy)
	if err != nil {
		return err
	}

	// Need encapsulation as the response from the GET is different than the one in the PUT
	bodyStringNormalized, _ := structure.NormalizeJsonString(fmt.Sprintf("{\"policy\": %+s}", string(bodyString)))

	if err := d.Set("policy_id", policyResponse.PolicyID); err != nil {
		return fmt.Errorf("error setting policy_id: %s", err)
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

func resourceOpensearchISMPolicyUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutISMPolicy(d, m); err != nil {
		return err
	}

	return resourceOpensearchISMPolicyRead(d, m)
}

func resourceOpensearchISMPolicyDelete(d *schema.ResourceData, m interface{}) error {
	policyID := d.Id()
	path := fmt.Sprintf("/_plugins/_ism/policies/%s", policyID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		// Build request
		req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
		if err != nil {
			return fmt.Errorf("error building DELETE request: %w", err)
		}

		resp, err = client.Client.Client.Perform(req)
		if err == nil && resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if err != nil {
		return fmt.Errorf("error deleting policy: %s: %w", path, err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting policy: received status code %d", resp.StatusCode)
}

func resourceOpensearchGetISMPolicy(policyID string, m interface{}) (GetPolicyResponse, error) {
	var err error
	response := new(GetPolicyResponse)

	path := fmt.Sprintf("/_plugins/_ism/policies/%s", policyID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return *response, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return *response, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return *response, fmt.Errorf("error getting policy: %s: %w", path, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return *response, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return *response, fmt.Errorf("policy not found: %s", policyID)
	}

	if resp.StatusCode != http.StatusOK {
		return *response, fmt.Errorf("error getting policy: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return *response, fmt.Errorf("error unmarshalling policy body: %+v: %+v", err, body)
	}

	normalizePolicy(response.Policy)

	return *response, err
}

func resourceOpensearchPutISMPolicy(d *schema.ResourceData, m interface{}) (*PutPolicyResponse, error) {
	response := new(PutPolicyResponse)
	policyJSON := d.Get("body").(string)
	seq := d.Get("seq_no").(int)
	primTerm := d.Get("primary_term").(int)

	policyID := d.Get("policy_id").(string)
	path := fmt.Sprintf("/_plugins/_ism/policies/%s", policyID)

	// Build query parameters
	var queryParts []string
	if seq >= 0 && primTerm > 0 {
		queryParts = append(queryParts, fmt.Sprintf("if_seq_no=%d", seq))
		queryParts = append(queryParts, fmt.Sprintf("if_primary_term=%d", primTerm))
	}
	if len(queryParts) > 0 {
		path = fmt.Sprintf("%s?%s", path, strings.Join(queryParts, "&"))
	}

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		// Build request (must recreate for each attempt as body can't be reused)
		req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(policyJSON))
		if err != nil {
			return response, fmt.Errorf("error building PUT request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Client.Client.Perform(req)
		if err == nil {
			// Check if we should retry
			if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
				break
			}
			resp.Body.Close()
		} else {
			// Request failed, will retry
			if attempt < maxRetries-1 {
				continue
			}
		}
	}

	if err != nil {
		return response, fmt.Errorf("error putting policy: %s: %w", path, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error putting policy: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling policy body: %+v: %+v", err, body)
	}

	return response, nil
}

type GetPolicyResponse struct {
	PolicyID    string                 `json:"_id"`
	Version     int                    `json:"_version"`
	PrimaryTerm int                    `json:"_primary_term"`
	SeqNo       int                    `json:"_seq_no"`
	Policy      map[string]interface{} `json:"policy"`
}

type PutPolicyResponse struct {
	PolicyID    string `json:"_id"`
	Version     int    `json:"_version"`
	PrimaryTerm int    `json:"_primary_term"`
	SeqNo       int    `json:"_seq_no"`
	Policy      struct {
		Policy map[string]interface{} `json:"policy"`
	} `json:"policy"`
}
