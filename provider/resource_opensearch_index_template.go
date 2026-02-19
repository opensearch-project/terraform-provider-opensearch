package provider

import (
	"context"
	"encoding/json"
	"log"
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
	res, err := client.Client.Template.Get(context.TODO(), &opensearchapi.TemplateGetReq{
		Templates: []string{id},
	})
	if err != nil {
		return "", err
	}

	// No more than 1 element is expected
	t, ok := res.Templates[id]
	if !ok {
		return "", nil
	}

	// For legacy templates, the API returns settings/mappings/aliases at top level
	// But the test config expects them nested under "template" key
	// We need to wrap them in a "template" section to match the expected format
	tplMap := map[string]interface{}{
		"index_patterns": t.IndexPatterns,
	}

	if t.Order != 0 {
		tplMap["order"] = t.Order
	}
	if t.Version != 0 {
		tplMap["version"] = t.Version
	}

	// Always include template section if any of settings/mappings/aliases exist
	templateSection := map[string]interface{}{}
	hasContent := false

	// Check settings - API returns {} even if empty
	if len(t.Settings) > 0 && string(t.Settings) != "{}" && string(t.Settings) != "null" {
		var settings map[string]interface{}
		if err := json.Unmarshal(t.Settings, &settings); err == nil && len(settings) > 0 {
			templateSection["settings"] = settings
			hasContent = true
		}
	}
	// Check mappings
	if len(t.Mappings) > 0 && string(t.Mappings) != "{}" && string(t.Mappings) != "null" {
		var mappings map[string]interface{}
		if err := json.Unmarshal(t.Mappings, &mappings); err == nil && len(mappings) > 0 {
			templateSection["mappings"] = mappings
			hasContent = true
		}
	}
	// Check aliases
	if len(t.Aliases) > 0 && string(t.Aliases) != "{}" && string(t.Aliases) != "null" {
		var aliases map[string]interface{}
		if err := json.Unmarshal(t.Aliases, &aliases); err == nil && len(aliases) > 0 {
			templateSection["aliases"] = aliases
			hasContent = true
		}
	}

	// Always include template section if user provided one in config
	// The diff suppression will handle empty vs non-empty comparisons
	if hasContent {
		tplMap["template"] = templateSection
	}

	normalizeIndexTemplate(tplMap)

	tj, err := json.Marshal(tplMap)
	if err != nil {
		return "", err
	}

	return string(tj), nil
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
