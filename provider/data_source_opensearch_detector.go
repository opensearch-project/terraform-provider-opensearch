package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	elastic7 "github.com/olivere/elastic/v7"
)

func dataSourceOpensearchDetector() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get information about an OpenSearch Security Analytics detector.",
		Read:        dataSourceOpensearchDetectorRead,

		Schema: map[string]*schema.Schema{
			"detector_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the detector to retrieve.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the detector to search for.",
			},
			"detector_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The detector type to filter by.",
			},
			// Output attributes
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the detector is enabled.",
			},
			"schedule": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The schedule configuration of the detector.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Details for the frequency of the schedule.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"interval": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The interval at which the detector runs.",
									},
									"unit": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The interval's unit of time.",
									},
								},
							},
						},
					},
				},
			},
			"inputs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The detector inputs configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detector_input": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The detector input configuration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"description": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Description of the detector.",
									},
									"indices": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "The log data source used for the detector.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"custom_rules": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Detector inputs for custom rules.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"id": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "A valid rule ID for the custom rule.",
												},
											},
										},
									},
									"pre_packaged_rules": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Detector inputs for pre-packaged rules.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"id": {
													Type:        schema.TypeString,
													Computed:    true,
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
				Computed:    true,
				Description: "Trigger settings for alerts.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique ID for the trigger.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the trigger.",
						},
						"severity": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Severity level for the trigger.",
						},
						"types": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Types for the trigger.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"ids": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "A list of rule IDs that become part of the trigger condition.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"sev_levels": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Sigma rule severity levels.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"tags": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Tags to focus the trigger conditions for alerts.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"detection_types": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Detection types for the trigger.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"actions": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Actions send notifications when trigger conditions are met.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Unique ID for the action.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Name for the trigger alert.",
									},
									"destination_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Unique ID for the notification destination.",
									},
									"subject_template": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Contains the information for the subject field of the notification message.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"source": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The subject for the notification message.",
												},
												"lang": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The scripting language used to define the subject.",
												},
											},
										},
									},
									"message_template": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Contains the information for the body of the notification message.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"source": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The body of the notification message.",
												},
												"lang": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The scripting language used to define the message.",
												},
											},
										},
									},
									"throttle_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether throttling is enabled for alert notifications.",
									},
									"throttle": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Throttling configuration.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"unit": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "Unit of time for throttling.",
												},
												"value": {
													Type:        schema.TypeInt,
													Computed:    true,
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
		},
	}
}

func dataSourceOpensearchDetectorRead(d *schema.ResourceData, m interface{}) error {
	detectorID := d.Get("detector_id").(string)
	detectorName := d.Get("name").(string)
	detectorType := d.Get("detector_type").(string)

	var detector *detectorSearchHit
	var err error

	if detectorID != "" {
		// Direct lookup by ID
		detector, err = dataSourceOpensearchGetDetectorByID(detectorID, m)
		if err != nil {
			return err
		}
	} else {
		// Search by name or type
		detector, err = dataSourceOpensearchSearchDetector(detectorName, detectorType, m)
		if err != nil {
			return err
		}
	}

	if detector == nil {
		return fmt.Errorf("detector not found")
	}

	d.SetId(detector.ID)
	d.Set("name", detector.Source.Name)
	d.Set("detector_type", detector.Source.DetectorType)
	
	// Flatten the detector data using the same function as the resource
	return flattenDetector(d, detector.Source.Detector)
}

func dataSourceOpensearchGetDetectorByID(detectorID string, m interface{}) (*detectorSearchHit, error) {
	res, err := resourceOpensearchGetDetector(detectorID, m)
	if err != nil {
		return nil, err
	}

	// Convert detectorResponse to detectorSearchHit format
	detector := &detectorSearchHit{
		ID: res.ID,
		Source: detectorSearchSource{
			Name:        res.Detector["name"].(string),
			DetectorType: res.Detector["detector_type"].(string),
			Detector:    res.Detector,
		},
	}

	return detector, nil
}

func dataSourceOpensearchSearchDetector(name, detectorType string, m interface{}) (*detectorSearchHit, error) {
	searchBody := map[string]interface{}{
		"size": 1,
		"query": map[string]interface{}{
			"nested": map[string]interface{}{
				"path": "detector",
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{},
					},
				},
			},
		},
	}

	// Build query conditions
	conditions := []interface{}{}
	
	if name != "" {
		conditions = append(conditions, map[string]interface{}{
			"match": map[string]interface{}{
				"detector.name": name,
			},
		})
	}
	
	if detectorType != "" {
		conditions = append(conditions, map[string]interface{}{
			"match": map[string]interface{}{
				"detector.detector_type": detectorType,
			},
		})
	}

	// If no conditions specified, match all
	if len(conditions) == 0 {
		conditions = append(conditions, map[string]interface{}{
			"match_all": map[string]interface{}{},
		})
	}

	searchBody["query"].(map[string]interface{})["nested"].(map[string]interface{})["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = conditions

	searchJSON, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling search body: %+v", err)
	}

	path := "/_plugins/_security_analytics/detectors/_search"
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	var body json.RawMessage
	res, err := osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(searchJSON),
	})
	if err != nil {
		return nil, err
	}
	body = res.Body

	var searchResponse detectorSearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling search response: %+v: %+v", err, body)
	}

	log.Printf("[INFO] Search response: %+v", searchResponse)

	if len(searchResponse.Hits.Hits) == 0 {
		return nil, fmt.Errorf("no detectors found matching criteria")
	}

	detector := searchResponse.Hits.Hits[0]
	
	// The search response has a different structure, need to adapt
	if detector.Source.Type == "detector" {
		// Build the detector object from the search response
		detectorData := map[string]interface{}{
			"name":         detector.Source.Name,
			"detector_type": detector.Source.DetectorType,
			"enabled":       detector.Source.Enabled,
			"schedule":      detector.Source.Schedule,
			"inputs":        detector.Source.Inputs,
			"triggers":      detector.Source.Triggers,
		}
		
		if detector.Source.LastUpdateTime != 0 {
			detectorData["last_update_time"] = fmt.Sprintf("%d", detector.Source.LastUpdateTime)
		}
		
		if detector.Source.EnabledTime != 0 {
			detectorData["enabled_time"] = fmt.Sprintf("%d", detector.Source.EnabledTime)
		}
		
		detector.Source.Detector = detectorData
	}

	return &detector, nil
}

type detectorSearchResponse struct {
	Hits detectorSearchHits `json:"hits"`
}

type detectorSearchHits struct {
	Hits []detectorSearchHit `json:"hits"`
}

type detectorSearchHit struct {
	ID     string                `json:"_id"`
	Source detectorSearchSource `json:"_source"`
}

type detectorSearchSource struct {
	Type           string                 `json:"type"`
	Name           string                 `json:"name"`
	DetectorType   string                 `json:"detector_type"`
	Enabled        bool                   `json:"enabled"`
	EnabledTime    int64                  `json:"enabled_time"`
	Schedule       interface{}            `json:"schedule"`
	Inputs         interface{}            `json:"inputs"`
	Triggers       interface{}            `json:"triggers"`
	LastUpdateTime int64                  `json:"last_update_time"`
	Detector       map[string]interface{} `json:"detector"`
}