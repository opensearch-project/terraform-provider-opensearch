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

func resourceOpensearchComposableIndexTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchComposableIndexTemplateCreate,
		Read:   resourceOpensearchComposableIndexTemplateRead,
		Update: resourceOpensearchComposableIndexTemplateUpdate,
		Delete: resourceOpensearchComposableIndexTemplateDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The name of the index template.",
			},
			"body": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: diffSuppressComposableIndexTemplate,
				ValidateFunc:     validation.StringIsJSON,
				Description:      "The JSON body of the index template.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Provides an Composable index template resource. This resource uses the `/_index_template` endpoint of the API that is available since version 2.0.0. Use `opensearch_index_template` if you are using older versions or if you want to keep using legacy Index Templates.",
	}
}

func resourceOpensearchComposableIndexTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceOpensearchPutComposableIndexTemplate(d, meta, true)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceOpensearchComposableIndexTemplateRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var result string

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}
	result, err = getIndexTemplate(client, id)
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

func getIndexTemplate(client *OpenSearchClient, id string) (string, error) {
	res, err := client.Client.IndexTemplate.Get(context.TODO(), &opensearchapi.IndexTemplateGetReq{
		IndexTemplates: []string{id},
	})
	log.Printf("[INFO] Index template %+v %+v", res, err)
	if err != nil {
		return "", err
	}

	// No more than 1 element is expected, if the index template is not found, previous call should
	// return a 404 error
	if len(res.IndexTemplates) == 0 {
		return "", nil
	}

	t := res.IndexTemplates[0].IndexTemplate
	tj, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(tj), nil
}

func resourceOpensearchComposableIndexTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceOpensearchPutComposableIndexTemplate(d, meta, false)
}

func resourceOpensearchComposableIndexTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	_, err = client.Client.IndexTemplate.Delete(context.TODO(), opensearchapi.IndexTemplateDeleteReq{
		IndexTemplate: id,
	})

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchPutComposableIndexTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	_, err = client.Client.IndexTemplate.Create(context.TODO(), opensearchapi.IndexTemplateCreateReq{
		IndexTemplate: name,
		Body:          strings.NewReader(body),
	})

	return err
}
