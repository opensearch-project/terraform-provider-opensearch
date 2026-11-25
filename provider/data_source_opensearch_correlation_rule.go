package provider

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpensearchCorrelationRule() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve information about an existing OpenSearch correlation rule.",
		Read:        dataSourceOpensearchCorrelationRuleRead,

		Schema: map[string]*schema.Schema{
			"rule_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the correlation rule to retrieve.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the correlation rule.",
			},
			"correlate": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of correlation queries.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"index": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the index used as the log source.",
						},
						"query": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The query used to filter security logs for correlation.",
						},
						"category": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The log type associated with the log source.",
						},
						"field": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Optional field to group correlations by.",
						},
					},
				},
			},
			"time_window": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Time window in milliseconds within which correlations must occur.",
			},
			"trigger": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Alert trigger configuration for the correlation rule.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the trigger.",
						},
						"severity": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Severity level (1-5, where 1 is highest).",
						},
						"actions": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Actions to execute when correlation is detected.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Name of the action.",
									},
									"destination_id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "ID of the notification channel.",
									},
									"subject_template": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Subject template for the notification.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"source": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "Template source.",
												},
												"lang": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "Template language.",
												},
											},
										},
									},
									"message_template": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Message template for the notification.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"source": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "Template source.",
												},
												"lang": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "Template language.",
												},
											},
										},
									},
									"throttle_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether throttling is enabled for this action.",
									},
									"throttle": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Throttle configuration for the action.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"unit": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "Time unit for throttle period.",
												},
												"value": {
													Type:        schema.TypeInt,
													Computed:    true,
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
		},
	}
}

func dataSourceOpensearchCorrelationRuleRead(d *schema.ResourceData, m interface{}) error {
	ruleID := d.Get("rule_id").(string)

	res, err := dataSourceGetCorrelationRule(ruleID, m)
	if err != nil {
		return err
	}

	d.SetId(ruleID)

	ds := &resourceDataSetter{d: d}
	ds.set("name", res.Rule.Name)
	ds.set("correlate", flattenCorrelate(res.Rule.Correlate))
	ds.set("time_window", res.Rule.TimeWindow)
	ds.set("trigger", flattenTrigger(res.Rule.Trigger))

	return ds.err
}

func dataSourceGetCorrelationRule(ruleID string, m interface{}) (*correlationRuleResponse, error) {
	// Use the shared search-based retrieval function
	response, err := getCorrelationRuleBySearch(ruleID, m)
	if err != nil {
		return response, err
	}

	log.Printf("[INFO] Retrieved correlation rule: %s", ruleID)
	return response, nil
}
