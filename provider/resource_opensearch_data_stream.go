package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/olivere/elastic/uritemplates"
	elastic7 "github.com/olivere/elastic/v7"
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
	osClient, err := getClient(providerConf)
	if err != nil {
		return err
	}
	err = elastic7GetDataStream(osClient, id)
	if err != nil {
		if elastic7.IsNotFound(err) {
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
	osClient, err := getClient(providerConf)
	if err != nil {
		return err
	}

	err = elastic7DeleteDataStream(osClient, id)

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceOpensearchPutDataStream(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)

	providerConf := meta.(*ProviderConf)
	osClient, err := getClient(providerConf)
	if err != nil {
		return err
	}

	err = elastic7PutDataStream(osClient, name)

	return err
}

func elastic7GetDataStream(client *elastic7.Client, id string) error {
	path, err := uritemplates.Expand("/_data_stream/{id}", map[string]string{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("error building URL path for data stream: %+v", err)
	}

	_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	return err
}

func elastic7DeleteDataStream(client *elastic7.Client, id string) error {
	path, err := uritemplates.Expand("/_data_stream/{id}", map[string]string{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("error building URL path for data stream: %+v", err)
	}

	_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "DELETE",
		Path:   path,
	})
	return err
}

func elastic7PutDataStream(client *elastic7.Client, id string) error {
	path, err := uritemplates.Expand("/_data_stream/{id}", map[string]string{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("error building URL path for data stream: %+v", err)
	}

	_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "PUT",
		Path:   path,
	})
	return err
}
