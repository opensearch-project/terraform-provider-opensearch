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

// validateSingleIndex ensures that exactly one index is provided in the indices field
func validateSingleIndex(v interface{}, k string) (warnings []string, errors []error) {
	indices := v.([]interface{})
	
	if len(indices) == 0 {
		errors = append(errors, fmt.Errorf("%s must contain exactly one index, but got 0", k))
		return warnings, errors
	}
	
	if len(indices) > 1 {
		errors = append(errors, fmt.Errorf("%s must contain exactly one index, but got %d. OpenSearch Security Analytics detectors currently support only one index per detector input", k, len(indices)))
		return warnings, errors
	}
	
	// Validate that the single index is not empty
	if indices[0] == nil || indices[0].(string) == "" {
		errors = append(errors, fmt.Errorf("%s cannot contain empty index names", k))
		return warnings, errors
	}
	
	return warnings, errors
}

func resourceOpenSearchDetector() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Security Analytics detector. Please refer to the OpenSearch Security Analytics detector documentation for details.",
		Create:      resourceOpensearchDetectorCreate,
		Read:        resourceOpensearchDetectorRead,
		Update:      resourceOpensearchDetectorUpdate,
		Delete:      resourceOpensearchDetectorDelete,
		Schema:      detectorSchema(),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func detectorSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "Name of the detector. Must be between 5 and 50 characters and consist of upper and lowercase letters, numbers 0-9, hyphens, spaces, and underscores.",
			ValidateFunc: validation.StringLenBetween(5, 50),
		},
		"detector_type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The log type that defines the detector.",
			ValidateFunc: validation.StringInSlice([]string{
				"linux", "network", "windows", "ad_ldap", "apache_access",
				"cloudtrail", "dns", "s3", "openstack_keystone_api",
			}, false),
		},
		"enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Sets the detector as either active (true) or inactive (false).",
		},
		"schedule": {
			Type:        schema.TypeList,
			Required:    true,
			MaxItems:    1,
			Description: "The schedule that determines how often the detector runs.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"period": {
						Type:        schema.TypeList,
						Required:    true,
						MaxItems:    1,
						Description: "Details for the frequency of the schedule.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"interval": {
									Type:        schema.TypeInt,
									Required:    true,
									Description: "The interval at which the detector runs.",
								},
								"unit": {
									Type:        schema.TypeString,
									Required:    true,
									Description: "The interval's unit of time.",
									ValidateFunc: validation.StringInSlice([]string{
										"MINUTES", "HOURS", "DAYS",
									}, false),
								},
							},
						},
					},
				},
			},
		},
		"inputs": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "Detector inputs.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"detector_input": {
						Type:        schema.TypeList,
						Required:    true,
						MaxItems:    1,
						Description: "The detector input configuration.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"description": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "Description of the detector.",
								},
								"indices": {
									Type:        schema.TypeList,
									Required:    true,
									Description: "The log data source used for the detector. Currently only one index is supported.",
									ValidateFunc: validateSingleIndex,
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"custom_rules": {
									Type:        schema.TypeList,
									Optional:    true,
									Description: "Detector inputs for custom rules.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"id": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "A valid rule ID for the custom rule.",
											},
										},
									},
								},
								"pre_packaged_rules": {
									Type:        schema.TypeList,
									Optional:    true,
									Description: "Detector inputs for pre-packaged rules.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"id": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "The rule ID for pre-packaged rules.",
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
		"triggers": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Trigger settings for alerts.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The unique ID for the trigger.",
					},
					"name": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  "The name of the trigger.",
						ValidateFunc: validation.StringLenBetween(5, 50),
					},
					"severity": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  "Severity level for the trigger.",
						ValidateFunc: validation.StringInSlice([]string{"1", "2", "3", "4", "5"}, false),
					},
					"types": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "Types for the trigger.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"ids": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "A list of rule IDs that become part of the trigger condition.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"sev_levels": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "Sigma rule severity levels.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{
								"informational", "low", "medium", "high", "critical",
							}, false),
						},
					},
					"tags": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "Tags to focus the trigger conditions for alerts.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"detection_types": {
						Type:        schema.TypeList,
						Optional:    true,
						Computed:    true,
						Description: "Detection types for the trigger. Defaults to [\"rules\"] if not specified.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						DefaultFunc: func() (interface{}, error) {
							return []interface{}{"rules"}, nil
						},
					},
					"actions": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "Actions send notifications when trigger conditions are met.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"id": {
									Type:        schema.TypeString,
									Required:    true,
									Description: "Unique ID for the action.",
								},
								"name": {
									Type:         schema.TypeString,
									Required:     true,
									Description:  "Name for the trigger alert.",
									ValidateFunc: validation.StringLenBetween(5, 50),
								},
								"destination_id": {
									Type:        schema.TypeString,
									Required:    true,
									Description: "Unique ID for the notification destination.",
								},
								"subject_template": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Contains the information for the subject field of the notification message.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"source": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "The subject for the notification message.",
											},
											"lang": {
												Type:         schema.TypeString,
												Required:     true,
												Description:  "The scripting language used to define the subject. Must be Mustache.",
												ValidateFunc: validation.StringInSlice([]string{"mustache"}, false),
											},
										},
									},
								},
								"message_template": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Contains the information for the body of the notification message.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"source": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "The body of the notification message.",
											},
											"lang": {
												Type:         schema.TypeString,
												Required:     true,
												Description:  "The scripting language used to define the message. Must be Mustache.",
												ValidateFunc: validation.StringInSlice([]string{"mustache"}, false),
											},
										},
									},
								},
								"throttle_enabled": {
									Type:        schema.TypeBool,
									Optional:    true,
									Default:     false,
									Description: "Enables throttling for alert notifications.",
								},
								"throttle": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Throttling limits the number of notifications you receive within a given span of time.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"unit": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "Unit of time for throttling.",
												ValidateFunc: validation.StringInSlice([]string{
													"MINUTES", "HOURS", "DAYS",
												}, false),
											},
											"value": {
												Type:        schema.TypeInt,
												Required:    true,
												Description: "The value for the unit of time.",
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
		"last_update_time": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Date and time when the detector was last updated.",
		},
		"enabled_time": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Date and time when the detector was last enabled.",
		},
		"version": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Version of the detector.",
		},
	}
}

func resourceOpensearchDetectorCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchPostDetector(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to create detector: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Detector ID: %s", d.Id())

	return resourceOpensearchDetectorRead(d, m)
}

func resourceOpensearchDetectorRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetDetector(d.Id(), m)

	if elastic7.IsNotFound(err) {
		log.Printf("[WARN] Detector (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	d.SetId(res.ID)
	
	flattenDetector(d, res.Detector)

	return nil
}

func resourceOpensearchDetectorUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchPutDetector(d, m)

	if err != nil {
		return err
	}

	return resourceOpensearchDetectorRead(d, m)
}

func resourceOpensearchDetectorDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_plugins/_security_analytics/detectors/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for detector: %+v", err)
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

func resourceOpensearchGetDetector(detectorID string, m interface{}) (*detectorResponse, error) {
	var err error
	response := new(detectorResponse)

	path, err := uritemplates.Expand("/_plugins/_security_analytics/detectors/{id}", map[string]string{
		"id": detectorID,
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for detector: %+v", err)
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
		return response, fmt.Errorf("error unmarshalling detector body: %+v: %+v", err, body)
	}
	
	log.Printf("[INFO] Detector response: %+v", response)
	return response, err
}

func resourceOpensearchPostDetector(d *schema.ResourceData, m interface{}) (*detectorResponse, error) {
	detectorBody := buildDetectorBody(d)
	detectorJSON, err := json.Marshal(detectorBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling detector body: %+v", err)
	}

	var response = new(detectorResponse)
	path := "/_plugins/_security_analytics/detectors"

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(detectorJSON),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling detector body: %+v: %+v", err, body)
	}
	
	return response, nil
}

func resourceOpensearchPutDetector(d *schema.ResourceData, m interface{}) (*detectorResponse, error) {
	detectorBody := buildDetectorBody(d)
	detectorJSON, err := json.Marshal(detectorBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling detector body: %+v", err)
	}

	var response = new(detectorResponse)

	path, err := uritemplates.Expand("/_plugins/_security_analytics/detectors/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for detector: %+v", err)
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
		Body:   string(detectorJSON),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling detector body: %+v: %+v", err, body)
	}

	return response, nil
}

func buildDetectorBody(d *schema.ResourceData) map[string]interface{} {
	body := map[string]interface{}{
		"name":         d.Get("name").(string),
		"detector_type": d.Get("detector_type").(string),
		"enabled":       d.Get("enabled").(bool),
		"type":          "detector",
	}

	// Schedule
	if v, ok := d.GetOk("schedule"); ok {
		scheduleList := v.([]interface{})
		if len(scheduleList) > 0 {
			schedule := scheduleList[0].(map[string]interface{})
			periodList := schedule["period"].([]interface{})
			if len(periodList) > 0 {
				period := periodList[0].(map[string]interface{})
				body["schedule"] = map[string]interface{}{
					"period": map[string]interface{}{
						"interval": period["interval"].(int),
						"unit":     period["unit"].(string),
					},
				}
			}
		}
	}

	// Inputs
	if v, ok := d.GetOk("inputs"); ok {
		inputsList := v.([]interface{})
		inputs := make([]interface{}, len(inputsList))
		
		for i, inputRaw := range inputsList {
			input := inputRaw.(map[string]interface{})
			detectorInputList := input["detector_input"].([]interface{})
			
			if len(detectorInputList) > 0 {
				detectorInput := detectorInputList[0].(map[string]interface{})
				inputBody := map[string]interface{}{}
				
				if desc, ok := detectorInput["description"]; ok && desc.(string) != "" {
					inputBody["description"] = desc.(string)
				}
				
				// Indices
				if indices, ok := detectorInput["indices"]; ok {
					indicesList := indices.([]interface{})
					inputBody["indices"] = make([]string, len(indicesList))
					for j, idx := range indicesList {
						inputBody["indices"].([]string)[j] = idx.(string)
					}
				}
				
				// Custom rules
				if customRules, ok := detectorInput["custom_rules"]; ok {
					customRulesList := customRules.([]interface{})
					if len(customRulesList) > 0 {
						rules := make([]interface{}, len(customRulesList))
						for j, ruleRaw := range customRulesList {
							rule := ruleRaw.(map[string]interface{})
							rules[j] = map[string]interface{}{
								"id": rule["id"].(string),
							}
						}
						inputBody["custom_rules"] = rules
					}
				}
				
				// Pre-packaged rules
				if prePackagedRules, ok := detectorInput["pre_packaged_rules"]; ok {
					prePackagedRulesList := prePackagedRules.([]interface{})
					if len(prePackagedRulesList) > 0 {
						rules := make([]interface{}, len(prePackagedRulesList))
						for j, ruleRaw := range prePackagedRulesList {
							rule := ruleRaw.(map[string]interface{})
							rules[j] = map[string]interface{}{
								"id": rule["id"].(string),
							}
						}
						inputBody["pre_packaged_rules"] = rules
					}
				}
				
				inputs[i] = map[string]interface{}{
					"detector_input": inputBody,
				}
			}
		}
		body["inputs"] = inputs
	}

	// Triggers
	if v, ok := d.GetOk("triggers"); ok {
		triggersList := v.([]interface{})
		triggers := make([]interface{}, len(triggersList))
		
		for i, triggerRaw := range triggersList {
			trigger := triggerRaw.(map[string]interface{})
			triggerBody := map[string]interface{}{
				"name":     trigger["name"].(string),
				"severity": trigger["severity"].(string),
			}
			
			// Optional fields
			if v, ok := trigger["types"]; ok {
				types := v.([]interface{})
				if len(types) > 0 {
					triggerBody["types"] = types
				} else {
					triggerBody["types"] = []interface{}{}
				}
			}
			
			if v, ok := trigger["ids"]; ok {
				ids := v.([]interface{})
				if len(ids) > 0 {
					triggerBody["ids"] = ids
				}
			}
			
			if v, ok := trigger["sev_levels"]; ok {
				sevLevels := v.([]interface{})
				if len(sevLevels) > 0 {
					triggerBody["sev_levels"] = sevLevels
				} else {
					triggerBody["sev_levels"] = []interface{}{}
				}
			}
			
			if v, ok := trigger["tags"]; ok {
				tags := v.([]interface{})
				if len(tags) > 0 {
					triggerBody["tags"] = tags
				}
			}
			
			if v, ok := trigger["detection_types"]; ok {
				detectionTypes := v.([]interface{})
				if len(detectionTypes) > 0 {
					triggerBody["detection_types"] = detectionTypes
				} else {
					// Set default detection_types to ["rules"] if empty
					triggerBody["detection_types"] = []interface{}{"rules"}
				}
			} else {
				// Set default detection_types to ["rules"] if not provided
				triggerBody["detection_types"] = []interface{}{"rules"}
			}
			
			// Actions
			if v, ok := trigger["actions"]; ok {
				actionsList := v.([]interface{})
				if len(actionsList) > 0 {
					actions := make([]interface{}, len(actionsList))
					for j, actionRaw := range actionsList {
						action := actionRaw.(map[string]interface{})
						actionBody := map[string]interface{}{
							"id":             action["id"].(string),
							"name":           action["name"].(string),
							"destination_id": action["destination_id"].(string),
							"throttle_enabled": action["throttle_enabled"].(bool),
						}
						
						// Subject template
						if subjectTemplate, ok := action["subject_template"].([]interface{}); ok && len(subjectTemplate) > 0 {
							st := subjectTemplate[0].(map[string]interface{})
							actionBody["subject_template"] = map[string]interface{}{
								"source": st["source"].(string),
								"lang":   st["lang"].(string),
							}
						}
						
						// Message template
						if messageTemplate, ok := action["message_template"].([]interface{}); ok && len(messageTemplate) > 0 {
							mt := messageTemplate[0].(map[string]interface{})
							actionBody["message_template"] = map[string]interface{}{
								"source": mt["source"].(string),
								"lang":   mt["lang"].(string),
							}
						}
						
						// Throttle
						if throttle, ok := action["throttle"].([]interface{}); ok && len(throttle) > 0 {
							t := throttle[0].(map[string]interface{})
							actionBody["throttle"] = map[string]interface{}{
								"unit":  t["unit"].(string),
								"value": t["value"].(int),
							}
						}
						
						actions[j] = actionBody
					}
					triggerBody["actions"] = actions
				} else {
					triggerBody["actions"] = []interface{}{}
				}
			}
			
			triggers[i] = triggerBody
		}
		body["triggers"] = triggers
	}

	return body
}

func flattenDetector(d *schema.ResourceData, detector map[string]interface{}) error {
	if name, ok := detector["name"].(string); ok {
		d.Set("name", name)
	}
	
	if detectorType, ok := detector["detector_type"].(string); ok {
		d.Set("detector_type", detectorType)
	}
	
	if enabled, ok := detector["enabled"].(bool); ok {
		d.Set("enabled", enabled)
	}
	
	if lastUpdateTime, ok := detector["last_update_time"].(string); ok {
		d.Set("last_update_time", lastUpdateTime)
	}
	
	if enabledTime, ok := detector["enabled_time"].(string); ok {
		d.Set("enabled_time", enabledTime)
	}

	// Schedule
	if schedule, ok := detector["schedule"].(map[string]interface{}); ok {
		if period, ok := schedule["period"].(map[string]interface{}); ok {
			scheduleList := []interface{}{
				map[string]interface{}{
					"period": []interface{}{
						map[string]interface{}{
							"interval": int(period["interval"].(float64)),
							"unit":     period["unit"].(string),
						},
					},
				},
			}
			d.Set("schedule", scheduleList)
		}
	}

	// Inputs
	if inputs, ok := detector["inputs"].([]interface{}); ok {
		inputsList := make([]interface{}, len(inputs))
		for i, inputRaw := range inputs {
			input := inputRaw.(map[string]interface{})
			if detectorInput, ok := input["detector_input"].(map[string]interface{}); ok {
				inputData := map[string]interface{}{}
				
				if desc, ok := detectorInput["description"].(string); ok {
					inputData["description"] = desc
				}
				
				if indices, ok := detectorInput["indices"].([]interface{}); ok {
					inputData["indices"] = indices
				}
				
				if customRules, ok := detectorInput["custom_rules"].([]interface{}); ok && len(customRules) > 0 {
					rules := make([]interface{}, len(customRules))
					for j, ruleRaw := range customRules {
						rule := ruleRaw.(map[string]interface{})
						rules[j] = map[string]interface{}{
							"id": rule["id"].(string),
						}
					}
					inputData["custom_rules"] = rules
				}
				
				if prePackagedRules, ok := detectorInput["pre_packaged_rules"].([]interface{}); ok && len(prePackagedRules) > 0 {
					rules := make([]interface{}, len(prePackagedRules))
					for j, ruleRaw := range prePackagedRules {
						rule := ruleRaw.(map[string]interface{})
						rules[j] = map[string]interface{}{
							"id": rule["id"].(string),
						}
					}
					inputData["pre_packaged_rules"] = rules
				}
				
				inputsList[i] = map[string]interface{}{
					"detector_input": []interface{}{inputData},
				}
			}
		}
		d.Set("inputs", inputsList)
	}

	// Triggers
	if triggers, ok := detector["triggers"].([]interface{}); ok && len(triggers) > 0 {
		triggersList := make([]interface{}, len(triggers))
		for i, triggerRaw := range triggers {
			trigger := triggerRaw.(map[string]interface{})
			triggerData := map[string]interface{}{
				"name":     trigger["name"].(string),
				"severity": trigger["severity"].(string),
			}
			
			if id, ok := trigger["id"].(string); ok {
				triggerData["id"] = id
			}
			
			if types, ok := trigger["types"].([]interface{}); ok {
				triggerData["types"] = types
			}
			
			if ids, ok := trigger["ids"].([]interface{}); ok {
				triggerData["ids"] = ids
			}
			
			if sevLevels, ok := trigger["sev_levels"].([]interface{}); ok {
				triggerData["sev_levels"] = sevLevels
			}
			
			if tags, ok := trigger["tags"].([]interface{}); ok {
				triggerData["tags"] = tags
			}
			
			if detectionTypes, ok := trigger["detection_types"].([]interface{}); ok {
				triggerData["detection_types"] = detectionTypes
			} else {
				// Ensure detection_types is set to default if not present in API response
				triggerData["detection_types"] = []interface{}{"rules"}
			}
			
			// Actions
			if actions, ok := trigger["actions"].([]interface{}); ok && len(actions) > 0 {
				actionsList := make([]interface{}, len(actions))
				for j, actionRaw := range actions {
					action := actionRaw.(map[string]interface{})
					actionData := map[string]interface{}{
						"id":               action["id"].(string),
						"name":             action["name"].(string),
						"destination_id":   action["destination_id"].(string),
						"throttle_enabled": action["throttle_enabled"].(bool),
					}
					
					if subjectTemplate, ok := action["subject_template"].(map[string]interface{}); ok {
						actionData["subject_template"] = []interface{}{
							map[string]interface{}{
								"source": subjectTemplate["source"].(string),
								"lang":   subjectTemplate["lang"].(string),
							},
						}
					}
					
					if messageTemplate, ok := action["message_template"].(map[string]interface{}); ok {
						actionData["message_template"] = []interface{}{
							map[string]interface{}{
								"source": messageTemplate["source"].(string),
								"lang":   messageTemplate["lang"].(string),
							},
						}
					}
					
					if throttle, ok := action["throttle"].(map[string]interface{}); ok {
						actionData["throttle"] = []interface{}{
							map[string]interface{}{
								"unit":  throttle["unit"].(string),
								"value": int(throttle["value"].(float64)),
							},
						}
					}
					
					actionsList[j] = actionData
				}
				triggerData["actions"] = actionsList
			}
			
			triggersList[i] = triggerData
		}
		d.Set("triggers", triggersList)
	}

	return nil
}

type detectorResponse struct {
	Version  int                    `json:"_version"`
	ID       string                 `json:"_id"`
	Detector map[string]interface{} `json:"detector"`
}