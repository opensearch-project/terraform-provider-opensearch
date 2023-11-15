package provider

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"
)

var maximumOSTemplateVersion, _ = version.NewVersion("1.0.0")

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

	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	result, err = elastic7IndexGetTemplate(osClient, id)
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
	res, err := client.IndexGetIndexTemplate(id).Do(context.TODO())
	log.Printf("[INFO] Index template %+v %+v", res, err)
	if err != nil {
		return "", err
	}

	// No more than 1 element is expected, if the index template is not found, previous call should
	// return a 404 error
	t := res.IndexTemplates[0].IndexTemplate
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
	log.Printf("[WARN] Index template (%s) will be deleted", id)
	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	err = elastic7IndexDeleteTemplate(osClient, id)

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
	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	openSearchVersion, err = version.NewVersion(providerConf.osVersion)
	if err == nil {
		err = elastic7IndexPutTemplate(openSearchVersion, osClient, name, body, create)
	}

	return err
}

func elastic7IndexPutTemplate(openSearchVersion *version.Version, client *elastic7.Client, name string, body string, create bool) error {
	var err error

	// making use of _template endpoint (legacy index templates) for older opensearch versions (<= 1.0.0)
	if openSearchVersion.LessThanOrEqual(maximumOSTemplateVersion) {
		_, err = client.IndexPutTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	} else {
		_, err = client.IndexPutIndexTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	}

	return err
}
