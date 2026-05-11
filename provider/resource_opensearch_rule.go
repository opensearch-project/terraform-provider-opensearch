package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
	"gopkg.in/yaml.v3"
)

var openSearchRuleSchema = map[string]*schema.Schema{
	"rule_id": {
		Description: "The ID of the rule.",
		Type:        schema.TypeString,
		Computed:    true,
	},
	"category": {
		Description: "The category of the rule (e.g., windows, linux, network).",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"rule": {
		Description:      "The Sigma rule in YAML format.",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: diffSuppressRule,
		ValidateFunc:     validateSigmaRule,
	},
	"forced": {
		Description: "Force the update/delete operation even if the rule is actively used by detectors. This is a request-level parameter and is not stored in state.",
		Type:        schema.TypeBool,
		Optional:    true,
	},
	"version": {
		Description: "The version of the rule.",
		Type:        schema.TypeInt,
		Computed:    true,
	},
}

func resourceOpenSearchRule() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Security Analytics custom rule. Please refer to the OpenSearch Security Analytics documentation for details.",
		Create:      resourceOpensearchRuleCreate,
		Read:        resourceOpensearchRuleRead,
		Update:      resourceOpensearchRuleUpdate,
		Delete:      resourceOpensearchRuleDelete,
		Schema:      openSearchRuleSchema,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			if d.HasChange("forced") && !d.HasChange("rule") && !d.HasChange("category") {
				return d.Clear("forced")
			}
			return nil
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchRuleCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchPostRule(d, m)
	if err != nil {
		log.Printf("[INFO] Failed to create rule: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Rule ID: %s", d.Id())

	return resourceOpensearchRuleRead(d, m)
}

func resourceOpensearchRuleRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetRule(d.Id(), m)

	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			log.Printf("[WARN] Rule (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	// Extract the rule content from the nested rule object
	var ruleContent string
	if ruleObj, ok := res.Rule["rule"]; ok {
		ruleContent = ruleObj.(string)
	} else {
		return fmt.Errorf("rule field not found in response")
	}

	// Convert from JSON back to YAML for Terraform state
	// Since we now send JSON to OpenSearch, it likely returns JSON format
	var ruleYAML string
	if strings.HasPrefix(strings.TrimSpace(ruleContent), "{") {
		// Content appears to be JSON, convert to YAML
		ruleYAML, err = convertJSONToYAML(ruleContent)
		if err != nil {
			// If conversion fails, try to use as-is (might be YAML already)
			log.Printf("[WARN] Failed to convert JSON to YAML, using raw content: %v", err)
			ruleYAML = ruleContent
		}
	} else {
		// Content appears to be YAML already
		ruleYAML = ruleContent
	}

	if err := d.Set("rule_id", res.ID); err != nil {
		return fmt.Errorf("error setting rule_id: %s", err)
	}
	if err := d.Set("rule", ruleYAML); err != nil {
		return fmt.Errorf("error setting rule: %s", err)
	}
	if err := d.Set("version", res.Version); err != nil {
		return fmt.Errorf("error setting version: %s", err)
	}

	// Set category from the response
	if category, ok := res.Rule["category"]; ok {
		if err := d.Set("category", category); err != nil {
			return fmt.Errorf("error setting category: %s", err)
		}
	}

	return nil
}

func resourceOpensearchRuleUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchPutRule(d, m)
	if err != nil {
		return err
	}

	return resourceOpensearchRuleRead(d, m)
}

func resourceOpensearchRuleDelete(d *schema.ResourceData, m interface{}) error {
	forced := d.Get("forced").(bool)

	path, err := uritemplates.Expand("/_plugins/_security_analytics/rules/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for rule: %+v", err)
	}

	// Add forced parameter if needed
	if forced {
		path = path + "?forced=true"
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

func resourceOpensearchGetRule(ruleID string, m interface{}) (*ruleResponse, error) {
	var err error
	response := new(ruleResponse)

	// Use search API to find the rule by ID
	path := "/_plugins/_security_analytics/rules/_search?pre_packaged=false"

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"_id": ruleID,
			},
		},
	}

	searchBody, err := json.Marshal(searchQuery)
	if err != nil {
		return response, fmt.Errorf("error marshalling search query: %+v", err)
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(searchBody),
	})
	if err != nil {
		return response, err
	}

	// Parse search response
	var searchResponse ruleSearchResponse
	if err := json.Unmarshal(res.Body, &searchResponse); err != nil {
		return response, fmt.Errorf("error unmarshalling search response: %+v: %+v", err, string(res.Body))
	}

	// Check if we found the rule
	if searchResponse.Hits.Total.Value == 0 {
		return response, fmt.Errorf("rule not found: %s", ruleID)
	}

	if len(searchResponse.Hits.Hits) == 0 {
		return response, fmt.Errorf("rule not found: %s", ruleID)
	}

	// Extract the first (and should be only) hit
	hit := searchResponse.Hits.Hits[0]
	response.ID = hit.ID
	response.Version = hit.Version
	response.Rule = hit.Source

	normalizeRule(response.Rule)
	log.Printf("[INFO] Response: %+v", response)

	return response, nil
}

func resourceOpensearchPostRule(d *schema.ResourceData, m interface{}) (*ruleResponse, error) {
	ruleYAML := d.Get("rule").(string)
	category := d.Get("category").(string)

	var err error
	response := new(ruleResponse)

	// Convert YAML to JSON before sending
	ruleJSON, err := convertYAMLToJSON(ruleYAML)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON: %v, ruleYAML: %s", err, ruleYAML)
	}

	path := fmt.Sprintf("/_plugins/_security_analytics/rules?category=%s", category)

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   ruleJSON,
	})
	if err != nil {
		return response, err
	}

	if err := json.Unmarshal(res.Body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling rule body: %+v: %+v", err, string(res.Body))
	}

	normalizeRule(response.Rule)
	return response, nil
}

func resourceOpensearchPutRule(d *schema.ResourceData, m interface{}) (*ruleResponse, error) {
	ruleYAML := d.Get("rule").(string)
	category := d.Get("category").(string)
	forced := d.Get("forced").(bool)

	var err error
	response := new(ruleResponse)

	// Convert YAML to JSON before sending
	ruleJSON, err := convertYAMLToJSON(ruleYAML)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON: %v", err)
	}

	path, err := uritemplates.Expand("/_plugins/_security_analytics/rules/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for rule: %+v", err)
	}

	// Add query parameters
	path = fmt.Sprintf("%s?category=%s", path, category)
	if forced {
		path = path + "&forced=true"
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "PUT",
		Path:   path,
		Body:   ruleJSON,
	})
	if err != nil {
		return response, err
	}

	if err := json.Unmarshal(res.Body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling rule body: %+v: %+v", err, string(res.Body))
	}

	return response, nil
}

// validateSigmaRule validates that the rule is valid YAML and contains required Sigma fields
func validateSigmaRule(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// Parse as YAML
	var rule map[string]interface{}
	if err := yaml.Unmarshal([]byte(value), &rule); err != nil {
		errors = append(errors, fmt.Errorf("%q must be valid YAML: %s", k, err))
		return
	}

	// Check for required Sigma rule fields
	requiredFields := []string{"title", "detection"}
	for _, field := range requiredFields {
		if _, ok := rule[field]; !ok {
			errors = append(errors, fmt.Errorf("%q must contain required Sigma field '%s'", k, field))
		}
	}

	// Validate logsource exists (can be empty but should be present)
	if _, ok := rule["logsource"]; !ok {
		errors = append(errors, fmt.Errorf("%q must contain 'logsource' field", k))
	}

	return
}

type ruleResponse struct {
	ID      string                 `json:"_id"`
	Version int                    `json:"_version"`
	Rule    map[string]interface{} `json:"rule"`
}

type ruleSearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID      string                 `json:"_id"`
			Version int                    `json:"_version"`
			Source  map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
