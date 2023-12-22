package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	elastic7 "github.com/olivere/elastic/v7"
)

func resourceOpensearchIngestPipeline() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch ingest pipeline resource.",
		Create:      resourceOpensearchIngestPipelineCreate,
		Read:        resourceOpensearchIngestPipelineRead,
		Update:      resourceOpensearchIngestPipelineUpdate,
		Delete:      resourceOpensearchIngestPipelineDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the ingest pipeline",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"body": {
				Description:      "The JSON body of the ingest pipeline",
				Type:             schema.TypeString,
				DiffSuppressFunc: diffSuppressIngestPipeline,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchIngestPipelineCreate(d *schema.ResourceData, meta interface{}) error {

	err := resourceOpensearchPutIngestPipeline(d, meta)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceOpensearchIngestPipelineRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var result string
	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	result, err = elastic7IngestGetPipeline(osClient, id)
	if err != nil {
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", d.Id())
	ds.set("body", result)
	return ds.err
}

func elastic7IngestGetPipeline(client *elastic7.Client, id string) (string, error) {

	res, err := client.IngestGetPipeline().Pretty(false).Do(context.TODO())
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

func resourceOpensearchIngestPipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceOpensearchPutIngestPipeline(d, meta)
}

func resourceOpensearchIngestPipelineDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osClient.IngestDeletePipeline(id).Do(context.TODO())

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchPutIngestPipeline(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	var err error
	osClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osClient.IngestPutPipeline(name).BodyString(body).Do(context.TODO())

	return err
}
