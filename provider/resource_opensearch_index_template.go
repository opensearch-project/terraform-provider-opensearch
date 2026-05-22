package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func resourceOpensearchIndexTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch index template resource.",
		Create:      resourceOpensearchIndexTemplateCreate,
		Read:        resourceOpensearchIndexTemplateRead,
		Update:      resourceOpensearchIndexTemplateUpdate,
		Delete:      resourceOpensearchIndexTemplateDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the index template.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"body": {
				Description:      "The JSON body of the index template.",
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: diffSuppressIndexTemplate,
				ValidateFunc:     validation.StringIsJSON,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchIndexTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceOpensearchPutIndexTemplate(d, meta, true)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceOpensearchIndexTemplateRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var result string
	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	result, err = getLegacyIndexTemplate(client, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] Index template (%s) not found, removing from state", id)
			d.SetId("")
			return nil
		}

		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", d.Id())
	ds.set("body", result)
	return ds.err
}

func getLegacyIndexTemplate(client *OpenSearchClient, id string) (string, error) {
	// Use direct HTTP request to get raw template data
	path := fmt.Sprintf("/_template/%s", id)

	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return "", fmt.Errorf("error building GET request: %w", err)
	}

	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting template: %d, %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var templateResponse map[string]json.RawMessage
	if err := json.Unmarshal(body, &templateResponse); err != nil {
		return "", err
	}

	// Extract the template data
	templateData, ok := templateResponse[id]
	if !ok {
		return "", nil
	}

	// Parse the template
	var tplMap map[string]interface{}
	if err := json.Unmarshal(templateData, &tplMap); err != nil {
		return "", err
	}

	// Normalize and return
	normalizeIndexTemplate(tplMap)

	result, err := json.Marshal(tplMap)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func resourceOpensearchIndexTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceOpensearchPutIndexTemplate(d, meta, false)
}

func resourceOpensearchIndexTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()
	log.Printf("[WARN] Index template (%s) will be deleted", id)
	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = client.Client.Template.Delete(context.TODO(), opensearchapi.TemplateDeleteReq{
		Template: id,
	})

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchPutIndexTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = client.Client.Template.Create(context.TODO(), opensearchapi.TemplateCreateReq{
		Template: name,
		Body:     strings.NewReader(body),
	})

	return err
}
