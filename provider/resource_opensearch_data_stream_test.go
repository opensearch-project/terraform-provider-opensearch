package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchDataStream(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)

		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchDataStreamDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDataStream,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDataStreamExists("opensearch_data_stream.foo"),
				),
			},
		},
	})
}

func testCheckOpensearchDataStreamExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No data stream ID is set")
		}

		meta := testAccProvider.Meta()

		var err error
		client, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		err = elastic7GetDataStream(client, rs.Primary.ID)
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchDataStreamDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_data_stream" {
			continue
		}

		meta := testAccProvider.Meta()

		var err error
		client, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		err = elastic7GetDataStream(client, rs.Primary.ID)

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Data stream %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchDataStream = `
resource "opensearch_composable_index_template" "foo" {
  name = "foo-template"
  body = <<EOF
{
  "index_patterns": ["foo-data-stream*"],
  "data_stream": {}
}
EOF
}

resource "opensearch_data_stream" "foo" {
  name       = "foo-data-stream"
  depends_on = [opensearch_composable_index_template.foo]
}
`
