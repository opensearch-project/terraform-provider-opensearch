package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAccOpensearchDataSourceHost_basic(t *testing.T) {
	var providers []*schema.Provider
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories(&providers),
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDataSourceHost,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_host.test", "id"),
					resource.TestCheckResourceAttrSet("data.opensearch_host.test", "url"),
				),
			},
		},
	})
}

var testAccOpensearchDataSourceHost = `
data "opensearch_host" "test" {
  active = true
}
`
