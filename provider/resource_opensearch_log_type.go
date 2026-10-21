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

func resourceOpenSearchLogType() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch custom log type. Please refer to the OpenSearch Security Analytics log type documentation for details.",
		Create:      resourceOpensearchLogTypeCreate,
		Read:        resourceOpensearchLogTypeRead,
		Update:      resourceOpensearchLogTypeUpdate,
		Delete:      resourceOpensearchLogTypeDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the custom log type.",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Description of the custom log type.",
			},
			"source": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Custom"}, false),
				Description:  "The source of the log type. Must be 'Custom' for user-created log types.",
			},
			"category": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Other",
				ValidateFunc: validation.StringInSlice([]string{
					"Other",
					"System Activity",
					"Network Activity",
					"Access Management",
					"Cloud Services",
					"Applications",
					"Security",
				}, false),
				Description: "The category of the log type. Valid values are: 'Other', 'System Activity', 'Network Activity', 'Access Management', 'Cloud Services', 'Applications', 'Security'. Defaults to 'Other'.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchLogTypeCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchPostLogType(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to create log type: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Log type created with ID: %s", d.Id())

	return resourceOpensearchLogTypeRead(d, m)
}

func resourceOpensearchLogTypeRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetLogType(d.Id(), m)

	if elastic7.IsNotFound(err) {
		log.Printf("[WARN] Log type (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", res.LogType.Name)
	ds.set("description", res.LogType.Description)
	ds.set("source", res.LogType.Source)
	ds.set("category", res.LogType.Category)

	return ds.err
}

func resourceOpensearchLogTypeUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchPutLogType(d, m)

	if err != nil {
		return err
	}

	return resourceOpensearchLogTypeRead(d, m)
}

func resourceOpensearchLogTypeDelete(d *schema.ResourceData, m interface{}) error {
	var err error

	path, err := uritemplates.Expand("/_plugins/_security_analytics/logtype/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for log type: %+v", err)
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

func resourceOpensearchGetLogType(logTypeID string, m interface{}) (*logTypeResponse, error) {
	// Build search query using map structure
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"_id": logTypeID,
			},
		},
	}

	path := "/_plugins/_security_analytics/logtype/_search"
	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("error marshalling search query: %+v", err)
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	res, err := osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(queryJSON),
	})
	if err != nil {
		return nil, err
	}

	var searchResponse logTypeSearchResponse
	if err := json.Unmarshal(res.Body, &searchResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling search response: %+v: %+v", err, res.Body)
	}

	// Check if we got any hits
	if searchResponse.Hits.Total.Value == 0 {
		return nil, &elastic7.Error{Status: 404}
	}

	// Get the first (and only) hit
	hit := searchResponse.Hits.Hits[0]

	// Transform search result into logTypeResponse format
	response := &logTypeResponse{
		ID:      hit.ID,
		Version: 0, // Version is not available in search response
	}
	response.LogType.Name = hit.Source.Name
	response.LogType.Description = hit.Source.Description
	response.LogType.Source = hit.Source.Source
	response.LogType.Category = hit.Source.Category
	if hit.Source.Tags != nil {
		response.LogType.Tags = hit.Source.Tags
	}

	return response, nil
}

func resourceOpensearchPostLogType(d *schema.ResourceData, m interface{}) (*logTypeResponse, error) {
	logTypeRequest := logTypeRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Source:      d.Get("source").(string),
		Category:    d.Get("category").(string),
	}

	logTypeJSON, err := json.Marshal(logTypeRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling log type: %+v", err)
	}

	var response = new(logTypeResponse)
	path := "/_plugins/_security_analytics/logtype"

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   string(logTypeJSON),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling log type body: %+v: %+v", err, body)
	}

	return response, nil
}

func resourceOpensearchPutLogType(d *schema.ResourceData, m interface{}) (*logTypeResponse, error) {
	logTypeRequest := logTypeRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Source:      d.Get("source").(string),
		Category:    d.Get("category").(string),
	}

	logTypeJSON, err := json.Marshal(logTypeRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling log type: %+v", err)
	}

	var response = new(logTypeResponse)

	path, err := uritemplates.Expand("/_plugins/_security_analytics/logtype/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for log type: %+v", err)
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
		Body:   string(logTypeJSON),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling log type body: %+v: %+v", err, body)
	}

	return response, nil
}

type logTypeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Category    string `json:"category,omitempty"`
}

type logTypeResponse struct {
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	LogType struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Source      string                 `json:"source"`
		Category    string                 `json:"category,omitempty"`
		Tags        map[string]interface{} `json:"tags,omitempty"`
	} `json:"logType"`
}

type logTypeSearchResponse struct {
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		Hits []struct {
			ID     string `json:"_id"`
			Source struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Source      string                 `json:"source"`
				Category    string                 `json:"category,omitempty"`
				Tags        map[string]interface{} `json:"tags,omitempty"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
