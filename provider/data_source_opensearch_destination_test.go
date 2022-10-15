package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccOpensearchDataSourceDestination_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDataSourceDestination,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_destination.test", "id"),
					resource.TestCheckResourceAttrSet("data.opensearch_destination.test", "body.type"),
				),
			},
		},
	})
}

var testAccOpensearchDataSourceDestination = `
resource "opensearch_destination" "test" {
  body = <<EOF
{
  "name": "my-destination",
  "type": "slack",
  "slack": {
    "url": "http://www.example.com"
  }
}
EOF
}

data "opensearch_destination" "test" {
  # Ugh, song and dance to get the json value to force dependency
  name = "${element(tolist(["my-destination", "${opensearch_destination.test.body}"]), 0)}"
}
`
