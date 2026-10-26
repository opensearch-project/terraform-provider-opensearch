package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchTransform(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed = true

	config := testAccOpensearchTransformV7

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)

			if !allowed {
				t.Skip("OpenSearch Transforms only supported on OpenSearch >= 1.1")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchTransformExists("opensearch_transform.test_transform"),
					resource.TestCheckResourceAttr(
						"opensearch_transform.test_transform",
						"transform_id",
						"test_transform",
					),
				),
			},
		},
	})
}

func testCheckOpensearchTransformExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No transform ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchGetTransform(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchTransformDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_transform" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		if err != nil {
			return err
		}
		_, err = resourceOpensearchGetTransform(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("OpensearchTransform %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchTransformV7 = `
resource "opensearch_index" "test" {
  name               = "terraform-test-transform-source"
  number_of_shards   = 1
  number_of_replicas = 0
}

resource "opensearch_transform" "test_transform" {
  transform_id = "test_transform"
  body         = <<EOF
  {
		"transform": {
			"enabled": true,
			"continuous": false,
			"description": "Test transform job",
			"schedule": {
				"interval": {
					"period": 1,
					"unit": "Minutes",
					"start_time": 1602100553
				}
			},
			"source_index": "${opensearch_index.test.name}",
			"target_index": "terraform-test-transform-target",
			"page_size": 10,
			"groups": [
				{
					"terms": {
						"source_field": "customer_gender",
						"target_field": "gender"
					}
				}
			],
			"aggregations": {
				"quantity": {
					"sum": {
						"field": "total_quantity"
					}
				}
			}
		}
	}
  EOF
}
`
