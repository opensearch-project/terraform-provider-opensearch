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

func resourceOpensearchComponentTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchComponentTemplateCreate,
		Read:   resourceOpensearchComponentTemplateRead,
		Update: resourceOpensearchComponentTemplateUpdate,
		Delete: resourceOpensearchComponentTemplateDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Name of the component template to create.",
			},
			"body": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: diffSuppressComponentTemplate,
				ValidateFunc:     validation.StringIsJSON,
				Description:      "The JSON body of the template.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Component templates are building blocks for constructing index templates that specify index mappings, settings, and aliases. You cannot directly apply a component template to a data stream or index. To be applied, a component template must be included in an index template's `composed_of` list.",
	}
}

func resourceOpensearchComponentTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceOpensearchPutComponentTemplate(d, meta, true)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceOpensearchComponentTemplateRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var result string

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	result, err = getComponentTemplate(client, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] Component template (%s) not found, removing from state", id)
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

func getComponentTemplate(client *OpenSearchClient, id string) (string, error) {
	res, err := client.Client.ComponentTemplate.Get(context.TODO(), &opensearchapi.ComponentTemplateGetReq{
		ComponentTemplate: id,
	})
	if err != nil {
		return "", err
	}

	// No more than 1 element is expected, if the component template is not found, previous call should
	// return a 404 error
	if len(res.ComponentTemplates) == 0 {
		return "", nil
	}

	t := res.ComponentTemplates[0].ComponentTemplate
	tj, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(tj), nil
}

func resourceOpensearchComponentTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceOpensearchPutComponentTemplate(d, meta, false)
}

func resourceOpensearchComponentTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	_, err = client.Client.ComponentTemplate.Delete(context.TODO(), opensearchapi.ComponentTemplateDeleteReq{
		ComponentTemplate: id,
	})

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchPutComponentTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	_, err = client.Client.ComponentTemplate.Create(context.TODO(), opensearchapi.ComponentTemplateCreateReq{
		ComponentTemplate: name,
		Body:              strings.NewReader(body),
	})

	return err
}
