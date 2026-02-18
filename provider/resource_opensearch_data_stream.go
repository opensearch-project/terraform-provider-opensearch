package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func resourceOpensearchDataStream() *schema.Resource {
	return &schema.Resource{
		Description: "A data stream lets you store append-only time series data across multiple (hidden, auto-generated) indices while giving you a single named resource for requests",
		Create:      resourceOpensearchDataStreamCreate,
		Read:        resourceOpensearchDataStreamRead,
		Delete:      resourceOpensearchDataStreamDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Name of the data stream to create, must have a matching ",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchDataStreamCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceOpensearchPutDataStream(d, meta)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return resourceOpensearchDataStreamRead(d, meta)
}

func resourceOpensearchDataStreamRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}
	err = getDataStream(client, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] data stream (%s) not found, removing from state", id)
			d.SetId("")
			return nil
		}

		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", d.Id())
	return ds.err
}

func resourceOpensearchDataStreamDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()
	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	_, err = client.Client.DataStream.Delete(context.TODO(), opensearchapi.DataStreamDeleteReq{
		DataStream: id,
	})

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchPutDataStream(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)

	providerConf := meta.(*ProviderConf)
	client, err := getOpenSearchClient(providerConf)
	if err != nil {
		return err
	}

	_, err = client.Client.DataStream.Create(context.TODO(), opensearchapi.DataStreamCreateReq{
		DataStream: name,
	})

	return err
}

func getDataStream(client *OpenSearchClient, id string) error {
	_, err := client.Client.DataStream.Get(context.TODO(), &opensearchapi.DataStreamGetReq{
		DataStreams: []string{id},
	})
	return err
}
