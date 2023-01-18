package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroDestination(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchOpenDistroDestination,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDestinationExists("opensearch_destination.test_destination"),
				),
			},
		},
	})
}

func TestAccOpensearchOpenDistroDestination_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchOpenDistroDestination,
			},
			{
				ResourceName:      "opensearch_destination.test_destination",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchDestinationExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No destination ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchOpenDistroQueryOrGetDestination(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchDestinationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_destination" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchOpenDistroQueryOrGetDestination(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Destination %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchOpenDistroDestination = `
resource "opensearch_destination" "test_destination" {
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
`
