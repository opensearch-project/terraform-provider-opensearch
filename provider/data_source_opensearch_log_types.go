package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	elastic7 "github.com/olivere/elastic/v7"
)

func dataSourceOpensearchLogTypes() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve information about OpenSearch log types. This includes both built-in log types (from Sigma) and custom user-created log types.",
		Read:        dataSourceOpensearchLogTypesRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by specific log type ID.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter log types by name (exact match).",
			},
			"source": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter log types by source (e.g., 'Custom', 'Sigma').",
			},
			"category": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter log types by category.",
			},
			"log_types": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of log types matching the filter criteria.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the log type.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the log type.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the log type.",
						},
						"source": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The source of the log type (e.g., 'Custom', 'Sigma').",
						},
						"category": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The category of the log type.",
						},
					},
				},
			},
		},
	}
}

func dataSourceOpensearchLogTypesRead(d *schema.ResourceData, m interface{}) error {
	// Build the search query based on provided filters
	searchQuery := buildLogTypesSearchQuery(d)

	path := "/_plugins/_security_analytics/logtype/_search"
	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return fmt.Errorf("error marshalling search query: %+v", err)
	}

	log.Printf("[DEBUG] Log types search query: %s", string(queryJSON))

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	res, err := osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(queryJSON),
	})
	if err != nil {
		return fmt.Errorf("error searching log types: %+v", err)
	}

	var searchResponse logTypeSearchResponse
	if err := json.Unmarshal(res.Body, &searchResponse); err != nil {
		return fmt.Errorf("error unmarshalling search response: %+v: %+v", err, string(res.Body))
	}

	log.Printf("[DEBUG] Found %d log types", searchResponse.Hits.Total.Value)

	// Convert hits to list of log types
	logTypes := make([]interface{}, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		logType := map[string]interface{}{
			"id":          hit.ID,
			"name":        hit.Source.Name,
			"description": hit.Source.Description,
			"source":      hit.Source.Source,
			"category":    hit.Source.Category,
		}
		logTypes = append(logTypes, logType)
	}

	if err := d.Set("log_types", logTypes); err != nil {
		return fmt.Errorf("error setting log_types: %+v", err)
	}

	// Generate a unique ID for this data source based on the filters
	d.SetId(generateLogTypesDataSourceID(d))

	return nil
}

// buildLogTypesSearchQuery builds an OpenSearch query based on the provided filters
func buildLogTypesSearchQuery(d *schema.ResourceData) map[string]interface{} {
	// Collect all filters that are set
	filters := make([]map[string]interface{}, 0)

	if id, ok := d.GetOk("id"); ok {
		filters = append(filters, map[string]interface{}{
			"match": map[string]interface{}{
				"_id": id.(string),
			},
		})
	}

	if name, ok := d.GetOk("name"); ok {
		filters = append(filters, map[string]interface{}{
			"match": map[string]interface{}{
				"name": name.(string),
			},
		})
	}

	if source, ok := d.GetOk("source"); ok {
		filters = append(filters, map[string]interface{}{
			"match": map[string]interface{}{
				"source": source.(string),
			},
		})
	}

	if category, ok := d.GetOk("category"); ok {
		filters = append(filters, map[string]interface{}{
			"match": map[string]interface{}{
				"category": category.(string),
			},
		})
	}

	// Build the appropriate query based on number of filters
	var query map[string]interface{}

	if len(filters) == 0 {
		// No filters, match all
		query = map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	} else if len(filters) == 1 {
		// Single filter, use it directly
		query = map[string]interface{}{
			"query": filters[0],
		}
	} else {
		// Multiple filters, use bool query with must clauses
		query = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": filters,
				},
			},
		}
	}

	return query
}

// generateLogTypesDataSourceID generates a unique ID for the data source based on filters
func generateLogTypesDataSourceID(d *schema.ResourceData) string {
	id := "log_types"

	if v, ok := d.GetOk("id"); ok {
		id += fmt.Sprintf("_id_%s", v.(string))
	}
	if v, ok := d.GetOk("name"); ok {
		id += fmt.Sprintf("_name_%s", v.(string))
	}
	if v, ok := d.GetOk("source"); ok {
		id += fmt.Sprintf("_source_%s", v.(string))
	}
	if v, ok := d.GetOk("category"); ok {
		id += fmt.Sprintf("_category_%s", v.(string))
	}

	return id
}
