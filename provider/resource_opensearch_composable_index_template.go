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

var minimalESComposableTemplateVersion, _ = version.NewVersion("7.8.0")

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
		Description: "Provides an Composable index template resource. This resource uses the `/_index_template` endpoint of the API that is available since version 7.8. Use `opensearch_index_template` if you are using older versions or if you want to keep using legacy Index Templates in versions 7.8+.",
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

func resourceOpensearchComposableIndexTemplateAvailable(v *version.Version, c *ProviderConf) bool {
	return v.GreaterThanOrEqual(minimalESComposableTemplateVersion) || c.flavor == Unknown
}

func resourceOpensearchComposableIndexTemplateRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var result string
	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	esClient, err := getClient(providerConf)
	if err != nil {
		return err
	}

	switch client := esClient.(type) {
	case *elastic7.Client:
		openSearchVersion, err = version.NewVersion(providerConf.osVersion)
		if err == nil {
			if resourceOpensearchComposableIndexTemplateAvailable(openSearchVersion, providerConf) {
				result, err = elastic7GetIndexTemplate(client, id)
			} else {
				err = fmt.Errorf("index_template endpoint only available from server version >= 7.8, got version %s", openSearchVersion.String())
			}
		}
	default:
		err = fmt.Errorf("index_template endpoint only available from server version >= 7.8, got version < 7.0.0")
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

func elastic7GetIndexTemplate(client *elastic7.Client, id string) (string, error) {
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

func resourceOpensearchComposableIndexTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceOpensearchPutComposableIndexTemplate(d, meta, false)
}

func resourceOpensearchComposableIndexTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	esClient, err := getClient(providerConf)
	if err != nil {
		return err
	}

	switch client := esClient.(type) {
	case *elastic7.Client:
		openSearchVersion, err = version.NewVersion(providerConf.osVersion)
		if err == nil {
			if resourceOpensearchComposableIndexTemplateAvailable(openSearchVersion, providerConf) {
				err = elastic7DeleteIndexTemplate(client, id)
			} else {
				err = fmt.Errorf("index_template endpoint only available from server version >= 7.8, got version %s", openSearchVersion.String())
			}
		}
	default:
		err = fmt.Errorf("index_template endpoint only available from server version >= 7.8, got version < 7.0.0")
	}

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func elastic7DeleteIndexTemplate(client *elastic7.Client, id string) error {
	_, err := client.IndexDeleteIndexTemplate(id).Do(context.TODO())
	return err
}

func resourceOpensearchPutComposableIndexTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	var openSearchVersion *version.Version

	providerConf := meta.(*ProviderConf)
	esClient, err := getClient(providerConf)
	if err != nil {
		return err
	}

	switch client := esClient.(type) {
	case *elastic7.Client:
		openSearchVersion, err = version.NewVersion(providerConf.osVersion)
		if err == nil {
			if resourceOpensearchComposableIndexTemplateAvailable(openSearchVersion, providerConf) {
				err = elastic7PutIndexTemplate(client, name, body, create)
			} else {
				err = fmt.Errorf("index_template endpoint only available from server version >= 7.8, got version %s", openSearchVersion.String())
			}
		}
	default:
		err = fmt.Errorf("index_template endpoint only available from server version >= 7.8, got version < 7.0.0")
	}

	return err
}

func elastic7PutIndexTemplate(client *elastic7.Client, name string, body string, create bool) error {
	_, err := client.IndexPutIndexTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	return err
}
