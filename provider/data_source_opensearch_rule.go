package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	elastic7 "github.com/olivere/elastic/v7"
)

func dataSourceOpensearchRule() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve information about OpenSearch Security Analytics rules.",
		Read:        dataSourceOpensearchRuleRead,

		Schema: map[string]*schema.Schema{
			"pre_packaged": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to search for pre-packaged rules (true) or custom rules (false).",
			},
			"category": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter rules by category (e.g., windows, linux, network).",
			},
			"log_source": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter rules by log source.",
			},
			"level": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter rules by severity level (critical, high, medium, low, informational).",
			},
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter rules by status (stable, test, experimental, deprecated, unsupported).",
			},
			"rules": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of rules matching the filters.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"rule_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the rule.",
						},
						"category": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The category of the rule.",
						},
						"title": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The title of the rule.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the rule.",
						},
						"level": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The severity level of the rule.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The status of the rule.",
						},
						"author": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The author of the rule.",
						},
						"log_source": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The log source of the rule.",
						},
						"rule": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The full Sigma rule in YAML format.",
						},
					},
				},
			},
		},
	}
}

func dataSourceOpensearchRuleRead(d *schema.ResourceData, m interface{}) error {
	prePackaged := d.Get("pre_packaged").(bool)
	category := d.Get("category").(string)
	logSource := d.Get("log_source").(string)
	level := d.Get("level").(string)
	status := d.Get("status").(string)

	// Build search query
	mustClauses := []map[string]interface{}{}

	if category != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"rule.category": category,
			},
		})
	}

	if logSource != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"rule.log_source": logSource,
			},
		})
	}

	if level != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"rule.level": level,
			},
		})
	}

	if status != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"rule.status": status,
			},
		})
	}

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"nested": map[string]interface{}{
				"path": "rule",
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": mustClauses,
					},
				},
			},
		},
		"from": 0,
		"size": 1000, // Fetch up to 1000 rules
	}

	// If no filters are provided, match all
	if len(mustClauses) == 0 {
		searchQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
			"from": 0,
			"size": 1000,
		}
	}

	searchBody, err := json.Marshal(searchQuery)
	if err != nil {
		return fmt.Errorf("error marshalling search query: %+v", err)
	}

	path := fmt.Sprintf("/_plugins/_security_analytics/rules/_search?pre_packaged=%t", prePackaged)

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(searchBody),
	})
	if err != nil {
		return fmt.Errorf("error searching rules: %+v", err)
	}

	// Parse search response
	var searchResponse ruleSearchResponse
	if err := json.Unmarshal(res.Body, &searchResponse); err != nil {
		return fmt.Errorf("error unmarshalling search response: %+v: %+v", err, string(res.Body))
	}

	// Build rules list
	rules := make([]map[string]interface{}, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		rule := make(map[string]interface{})
		rule["rule_id"] = hit.ID

		// Extract fields from the source
		if cat, ok := hit.Source["category"]; ok {
			rule["category"] = cat
		}
		if title, ok := hit.Source["title"]; ok {
			rule["title"] = title
		}
		if desc, ok := hit.Source["description"]; ok {
			rule["description"] = desc
		}
		if lvl, ok := hit.Source["level"]; ok {
			rule["level"] = lvl
		}
		if stat, ok := hit.Source["status"]; ok {
			rule["status"] = stat
		}
		if auth, ok := hit.Source["author"]; ok {
			rule["author"] = auth
		}
		if ls, ok := hit.Source["log_source"]; ok {
			rule["log_source"] = ls
		}
		if r, ok := hit.Source["rule"]; ok {
			rule["rule"] = r
		}

		rules = append(rules, rule)
	}

	if err := d.Set("rules", rules); err != nil {
		return fmt.Errorf("error setting rules: %s", err)
	}

	// Use a composite ID based on the search criteria
	id := fmt.Sprintf("rules_%t_%s_%s_%s_%s", prePackaged, category, logSource, level, status)
	d.SetId(id)

	log.Printf("[INFO] Found %d rules", len(rules))
	return nil
}
