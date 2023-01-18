package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"reflect"
)

func dataSourceOpensearchHost() *schema.Resource {
	return &schema.Resource{
		Description: "`opensearch_host` can be used to retrieve the host URL for the provider's current cluster.",
		Read:        dataSourceOpensearchHostRead,

		Schema: map[string]*schema.Schema{
			"active": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "should be set to `true`",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "the url of the active cluster",
			},
		},
	}
}

func dataSourceOpensearchHostRead(d *schema.ResourceData, m interface{}) error {

	// The upstream client does not export the property for the urls
	// it's using. Presumably the URLS would be available where the client is
	// intantiated, but in terraform, that's not always practicable.
	var err error
	client, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	var url string
	urls := reflect.ValueOf(client).Elem().FieldByName("urls")
	if urls.Len() > 0 {
		url = urls.Index(0).String()
	}

	d.SetId(url)
	err = d.Set("url", url)

	return err
}
