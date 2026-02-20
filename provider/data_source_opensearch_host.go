package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	// Get the URL directly from the provider configuration
	conf := m.(*ProviderConf)
	url := conf.rawUrl

	d.SetId(url)
	return d.Set("url", url)
}
