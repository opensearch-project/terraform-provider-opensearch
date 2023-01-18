package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	elastic7 "github.com/olivere/elastic/v7"
)

var esComponentTemplateMinimalVersion, _ = version.NewVersion("7.8.0")

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
		Description: "Component templates are building blocks for constructing index templates that specify index mappings, settings, and aliases. You cannot directly apply a component template to a data stream or index. To be applied, a component template must be included in an index template’s `composed_of` list.",
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
	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	client, err := getClient(providerConf)
	if err != nil {
		return err
	}

	openSearchVersion, err = version.NewVersion(providerConf.osVersion)
	if err == nil {
		if resourceOpensearchComponentTemplateAvailable(openSearchVersion, providerConf) {
			result, err = elastic7GetComponentTemplate(client, id)
		} else {
			err = fmt.Errorf("component_template endpoint only available from server version >= 7.8, got version %s", openSearchVersion.String())
		}
	}

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

func elastic7GetComponentTemplate(client *elastic7.Client, id string) (string, error) {
	res, err := client.IndexGetComponentTemplate(id).Do(context.TODO())
	if err != nil {
		return "", err
	}

	// No more than 1 element is expected, if the index template is not found, previous call should
	// return a 404 error
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

	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	client, err := getClient(providerConf)
	if err != nil {
		return err
	}

	openSearchVersion, err = version.NewVersion(providerConf.osVersion)
	if err == nil {
		if resourceOpensearchComponentTemplateAvailable(openSearchVersion, providerConf) {
			err = elastic7DeleteComponentTemplate(client, id)
		} else {
			err = fmt.Errorf("component_template endpoint only available from server version >= 7.8, got version %s", openSearchVersion.String())
		}
	}
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchComponentTemplateAvailable(v *version.Version, c *ProviderConf) bool {
	return v.GreaterThanOrEqual(esComponentTemplateMinimalVersion) || c.flavor == Unknown
}

func elastic7DeleteComponentTemplate(client *elastic7.Client, id string) error {
	_, err := client.IndexDeleteComponentTemplate(id).Do(context.TODO())
	return err
}

func resourceOpensearchPutComponentTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	client, err := getClient(providerConf)
	if err != nil {
		return err
	}

	openSearchVersion, err = version.NewVersion(providerConf.osVersion)
	if err == nil {
		if resourceOpensearchComponentTemplateAvailable(openSearchVersion, providerConf) {
			err = elastic7PutComponentTemplate(client, name, body, create)
		} else {
			err = fmt.Errorf("component_template endpoint only available from server version >= 7.8, got version %s", openSearchVersion.String())
		}
	}

	return err
}

func elastic7PutComponentTemplate(client *elastic7.Client, name string, body string, create bool) error {
	_, err := client.IndexPutComponentTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	return err
}
