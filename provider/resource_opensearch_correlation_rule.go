package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var opensearchCorrelationRuleSchema = map[string]*schema.Schema{
	"name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Name of the correlation rule.",
	},
	"correlate": {
		Type:        schema.TypeList,
		Required:    true,
		MinItems:    2,
		Description: "List of correlation queries to correlate findings across log sources.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"index": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The name of the index used as the log source.",
				},
				"query": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The query used to filter security logs for correlation.",
				},
				"category": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The log type associated with the log source.",
				},
				"field": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Optional field to group correlations by (e.g., user.id, source.ip).",
				},
			},
		},
	},
	"time_window": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "Time window in milliseconds within which correlations must occur.",
	},
	"trigger": {
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Alert trigger configuration for the correlation rule.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Name of the trigger.",
				},
				"severity": {
					Type:         schema.TypeString,
					Required:     true,
					Description:  "Severity level (1-5, where 1 is highest).",
					ValidateFunc: validation.StringInSlice([]string{"1", "2", "3", "4", "5"}, false),
				},
				"actions": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Actions to execute when correlation is detected.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Name of the action.",
							},
							"destination_id": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "ID of the notification channel.",
							},
							"subject_template": {
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Description: "Subject template for the notification.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"source": {
											Type:        schema.TypeString,
											Required:    true,
											Description: "Template source.",
										},
										"lang": {
											Type:        schema.TypeString,
											Required:    true,
											Description: "Template language (e.g., mustache).",
										},
									},
								},
							},
							"message_template": {
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Description: "Message template for the notification.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"source": {
											Type:        schema.TypeString,
											Required:    true,
											Description: "Template source.",
										},
										"lang": {
											Type:        schema.TypeString,
											Required:    true,
											Description: "Template language (e.g., mustache).",
										},
									},
								},
							},
							"throttle_enabled": {
								Type:        schema.TypeBool,
								Optional:    true,
								Default:     false,
								Description: "Whether throttling is enabled for this action.",
							},
							"throttle": {
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Description: "Throttle configuration for the action.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"unit": {
											Type:         schema.TypeString,
											Required:     true,
											Description:  "Time unit for throttle period.",
											ValidateFunc: validation.StringInSlice([]string{"MINUTES", "HOURS", "DAYS"}, false),
										},
										"value": {
											Type:        schema.TypeInt,
											Required:    true,
											Description: "Throttle period value.",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

func resourceOpenSearchCorrelationRule() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch correlation rule resource. Correlation rules allow you to correlate findings across multiple log sources to detect complex security threats.",
		Create:      resourceOpensearchCorrelationRuleCreate,
		Read:        resourceOpensearchCorrelationRuleRead,
		Update:      resourceOpensearchCorrelationRuleUpdate,
		Delete:      resourceOpensearchCorrelationRuleDelete,
		Schema:      opensearchCorrelationRuleSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchCorrelationRuleCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchPostCorrelationRule(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to create correlation rule: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Correlation rule created with ID: %s", d.Id())

	return resourceOpensearchCorrelationRuleRead(d, m)
}

func resourceOpensearchCorrelationRuleRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetCorrelationRule(d.Id(), m)

	if elastic7.IsNotFound(err) {
		log.Printf("[WARN] Correlation rule (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", res.Rule.Name)
	ds.set("correlate", flattenCorrelate(res.Rule.Correlate))
	ds.set("time_window", res.Rule.TimeWindow)
	ds.set("trigger", flattenTrigger(res.Rule.Trigger))

	return ds.err
}

func resourceOpensearchCorrelationRuleUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchPutCorrelationRule(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to update correlation rule: %+v", err)
		return err
	}

	return resourceOpensearchCorrelationRuleRead(d, m)
}

func resourceOpensearchCorrelationRuleDelete(d *schema.ResourceData, m interface{}) error {
	var err error

	path, err := uritemplates.Expand("/_plugins/_security_analytics/correlation/rules/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for correlation rule: %+v", err)
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

func resourceOpensearchGetCorrelationRule(ruleID string, m interface{}) (*correlationRuleResponse, error) {
	return getCorrelationRuleBySearch(ruleID, m)
}

// getCorrelationRuleBySearch retrieves a correlation rule by searching for it using the _search endpoint
func getCorrelationRuleBySearch(ruleID string, m interface{}) (*correlationRuleResponse, error) {
	response := new(correlationRuleResponse)

	path := "/_plugins/_security_analytics/correlation/rules/_search"
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"_id": ruleID,
			},
		},
	}

	searchJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return response, fmt.Errorf("error marshalling search query: %+v", err)
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	res, err := osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(searchJSON),
	})
	if err != nil {
		return response, fmt.Errorf("error searching for correlation rule: %w", err)
	}

	// Parse the search response
	var searchResponse correlationRuleSearchResponse
	if err := json.Unmarshal(res.Body, &searchResponse); err != nil {
		return response, fmt.Errorf("error unmarshalling search response: %+v: %+v", err, string(res.Body))
	}

	// Extract the first (and should be only) hit
	hit := searchResponse.Hits.Hits[0]
	response.ID = hit.ID
	response.Version = hit.Version
	response.Rule = hit.Source

	return response, nil
}

func resourceOpensearchPostCorrelationRule(d *schema.ResourceData, m interface{}) (*correlationRuleResponse, error) {
	var err error
	response := new(correlationRuleResponse)

	ruleBody := buildCorrelationRuleBody(d)

	ruleJSON, err := json.Marshal(ruleBody)
	if err != nil {
		return response, fmt.Errorf("error marshalling correlation rule body: %+v", err)
	}

	path := "/_plugins/_security_analytics/correlation/rules"

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(ruleJSON),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling correlation rule body: %+v: %+v", err, body)
	}

	return response, nil
}

func resourceOpensearchPutCorrelationRule(d *schema.ResourceData, m interface{}) (*correlationRuleResponse, error) {
	var err error
	response := new(correlationRuleResponse)

	ruleBody := buildCorrelationRuleBody(d)

	ruleJSON, err := json.Marshal(ruleBody)
	if err != nil {
		return response, fmt.Errorf("error marshalling correlation rule body: %+v", err)
	}

	path, err := uritemplates.Expand("/_plugins/_security_analytics/correlation/rules/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for correlation rule: %+v", err)
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
		Body:   string(ruleJSON),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling correlation rule body: %+v: %+v", err, body)
	}

	return response, nil
}

func buildCorrelationRuleBody(d *schema.ResourceData) map[string]interface{} {
	body := make(map[string]interface{})

	if v, ok := d.GetOk("name"); ok {
		body["name"] = v.(string)
	}

	if v, ok := d.GetOk("correlate"); ok {
		body["correlate"] = expandCorrelate(v.([]interface{}))
	}

	if v, ok := d.GetOk("time_window"); ok {
		body["time_window"] = v.(int)
	}

	if v, ok := d.GetOk("trigger"); ok {
		triggerList := v.([]interface{})
		if len(triggerList) > 0 {
			body["trigger"] = expandTrigger(triggerList[0].(map[string]interface{}))
		}
	}

	return body
}

func expandCorrelate(correlates []interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(correlates))

	for _, c := range correlates {
		correlate := c.(map[string]interface{})
		item := map[string]interface{}{
			"index":    correlate["index"].(string),
			"query":    correlate["query"].(string),
			"category": correlate["category"].(string),
		}

		if field, ok := correlate["field"].(string); ok && field != "" {
			item["field"] = field
		}

		result = append(result, item)
	}

	return result
}

func flattenCorrelate(correlates []correlationQuery) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(correlates))

	for _, correlate := range correlates {
		item := map[string]interface{}{
			"index":    correlate.Index,
			"query":    correlate.Query,
			"category": correlate.Category,
		}

		if correlate.Field != "" {
			item["field"] = correlate.Field
		}

		result = append(result, item)
	}

	return result
}

func expandTrigger(trigger map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"name":     trigger["name"].(string),
		"severity": trigger["severity"].(string),
	}

	if actions, ok := trigger["actions"].([]interface{}); ok && len(actions) > 0 {
		result["actions"] = expandActions(actions)
	}

	return result
}

func flattenTrigger(trigger *correlationTrigger) []map[string]interface{} {
	if trigger == nil {
		return nil
	}

	result := map[string]interface{}{
		"name":     trigger.Name,
		"severity": trigger.Severity,
	}

	if len(trigger.Actions) > 0 {
		result["actions"] = flattenActions(trigger.Actions)
	}

	return []map[string]interface{}{result}
}

func expandActions(actions []interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(actions))

	for _, a := range actions {
		action := a.(map[string]interface{})
		item := map[string]interface{}{
			"name":             action["name"].(string),
			"destination_id":   action["destination_id"].(string),
			"throttle_enabled": action["throttle_enabled"].(bool),
		}

		if subjectTemplate, ok := action["subject_template"].([]interface{}); ok && len(subjectTemplate) > 0 {
			st := subjectTemplate[0].(map[string]interface{})
			item["subject_template"] = map[string]interface{}{
				"source": st["source"].(string),
				"lang":   st["lang"].(string),
			}
		}

		if messageTemplate, ok := action["message_template"].([]interface{}); ok && len(messageTemplate) > 0 {
			mt := messageTemplate[0].(map[string]interface{})
			item["message_template"] = map[string]interface{}{
				"source": mt["source"].(string),
				"lang":   mt["lang"].(string),
			}
		}

		if throttle, ok := action["throttle"].([]interface{}); ok && len(throttle) > 0 {
			t := throttle[0].(map[string]interface{})
			item["throttle"] = map[string]interface{}{
				"unit":  t["unit"].(string),
				"value": t["value"].(int),
			}
		}

		result = append(result, item)
	}

	return result
}

func flattenActions(actions []correlationAction) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(actions))

	for _, action := range actions {
		item := map[string]interface{}{
			"name":             action.Name,
			"destination_id":   action.DestinationID,
			"throttle_enabled": action.ThrottleEnabled,
		}

		if action.SubjectTemplate != nil {
			item["subject_template"] = []map[string]interface{}{
				{
					"source": action.SubjectTemplate.Source,
					"lang":   action.SubjectTemplate.Lang,
				},
			}
		}

		if action.MessageTemplate != nil {
			item["message_template"] = []map[string]interface{}{
				{
					"source": action.MessageTemplate.Source,
					"lang":   action.MessageTemplate.Lang,
				},
			}
		}

		if action.Throttle != nil {
			item["throttle"] = []map[string]interface{}{
				{
					"unit":  action.Throttle.Unit,
					"value": action.Throttle.Value,
				},
			}
		}

		result = append(result, item)
	}

	return result
}

type correlationRuleResponse struct {
	ID      string          `json:"_id"`
	Version int             `json:"_version"`
	Rule    correlationRule `json:"rule"`
}

type correlationRule struct {
	Name       string              `json:"name,omitempty"`
	Correlate  []correlationQuery  `json:"correlate"`
	TimeWindow int                 `json:"time_window,omitempty"`
	Trigger    *correlationTrigger `json:"trigger,omitempty"`
}

type correlationQuery struct {
	Index    string `json:"index"`
	Query    string `json:"query"`
	Category string `json:"category"`
	Field    string `json:"field,omitempty"`
}

type correlationTrigger struct {
	ID       string              `json:"id,omitempty"`
	Name     string              `json:"name"`
	Severity string              `json:"severity"`
	Actions  []correlationAction `json:"actions,omitempty"`
}

type correlationAction struct {
	ID              string          `json:"id,omitempty"`
	Name            string          `json:"name"`
	DestinationID   string          `json:"destination_id"`
	SubjectTemplate *actionTemplate `json:"subject_template,omitempty"`
	MessageTemplate *actionTemplate `json:"message_template,omitempty"`
	ThrottleEnabled bool            `json:"throttle_enabled"`
	Throttle        *actionThrottle `json:"throttle,omitempty"`
}

type actionTemplate struct {
	Source string `json:"source"`
	Lang   string `json:"lang"`
}

type actionThrottle struct {
	Unit  string `json:"unit"`
	Value int    `json:"value"`
}

// Search response types
type correlationRuleSearchResponse struct {
	Took     int                       `json:"took"`
	TimedOut bool                      `json:"timed_out"`
	Shards   searchShards              `json:"_shards"`
	Hits     correlationRuleSearchHits `json:"hits"`
}

type searchShards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type correlationRuleSearchHits struct {
	Total    searchTotal                `json:"total"`
	MaxScore float64                    `json:"max_score"`
	Hits     []correlationRuleSearchHit `json:"hits"`
}

type searchTotal struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

type correlationRuleSearchHit struct {
	Index       string          `json:"_index"`
	ID          string          `json:"_id"`
	Version     int             `json:"_version"`
	SeqNo       int             `json:"_seq_no"`
	PrimaryTerm int             `json:"_primary_term"`
	Score       float64         `json:"_score"`
	Source      correlationRule `json:"_source"`
}
