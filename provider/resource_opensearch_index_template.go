package provider

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"
)

func resourceOpensearchIndexTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an Opensearch index template resource.",
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
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		result, err = elastic7IndexGetTemplate(client, id)
	case *elastic6.Client:
		result, err = elastic6IndexGetTemplate(client, id)
	default:
		return errors.New("opensearch version not supported")
	}
	if err != nil {
		if elastic7.IsNotFound(err) || elastic6.IsNotFound(err) {
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
	res, err := client.IndexGetTemplate(id).Do(context.TODO())
	if err != nil {
		return "", err
	}

	t := res[id]
	tj, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(tj), nil
}

func elastic6IndexGetTemplate(client *elastic6.Client, id string) (string, error) {
	res, err := client.IndexGetTemplate(id).Do(context.TODO())
	if err != nil {
		return "", err
	}

	t := res[id]
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

	var err error
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		err = elastic7IndexDeleteTemplate(client, id)
	case *elastic6.Client:
		err = elastic6IndexDeleteTemplate(client, id)
	default:
		return errors.New("opensearch version not supported")
	}

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func elastic7IndexDeleteTemplate(client *elastic7.Client, id string) error {
	_, err := client.IndexDeleteTemplate(id).Do(context.TODO())
	return err
}

func elastic6IndexDeleteTemplate(client *elastic6.Client, id string) error {
	_, err := client.IndexDeleteTemplate(id).Do(context.TODO())
	return err
}

func resourceOpensearchPutIndexTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	var err error
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		err = elastic7IndexPutTemplate(client, name, body, create)
	case *elastic6.Client:
		err = elastic6IndexPutTemplate(client, name, body, create)
	default:
		return errors.New("opensearch version not supported")
	}

	return err
}

func elastic7IndexPutTemplate(client *elastic7.Client, name string, body string, create bool) error {
	_, err := client.IndexPutTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	return err
}

func elastic6IndexPutTemplate(client *elastic6.Client, name string, body string, create bool) error {
	_, err := client.IndexPutTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	return err
}
