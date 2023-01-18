package provider

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	elastic7 "github.com/olivere/elastic/v7"
)

func resourceOpensearchIndexTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchIndexTemplateCreate,
		Read:   resourceOpensearchIndexTemplateRead,
		Update: resourceOpensearchIndexTemplateUpdate,
		Delete: resourceOpensearchIndexTemplateDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"body": {
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
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	result, err = elastic7IndexGetTemplate(client, id)

	if err != nil {
		if elastic7.IsNotFound(err) {
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

func elastic7IndexGetTemplate(client *elastic7.Client, id string) (string, error) {
	res, err := client.IndexGetIndexTemplate(id).Do(context.TODO())
	if err != nil {
		return "", err
	}

	t := res
	tj, err := json.Marshal(t)
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

	log.Printf("[WARN] Index template (%s) will be delete", id)

	var err error
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	err = elastic7IndexDeleteTemplate(client, id)

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func elastic7IndexDeleteTemplate(client *elastic7.Client, id string) error {
	_, err := client.IndexDeleteIndexTemplate(id).Do(context.TODO())
	return err
}

func resourceOpensearchPutIndexTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	var err error
	client, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	err = elastic7IndexPutTemplate(client, name, body, create)

	return err
}

func elastic7IndexPutTemplate(client *elastic7.Client, name string, body string, create bool) error {
	_, err := client.IndexPutIndexTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	return err
}
