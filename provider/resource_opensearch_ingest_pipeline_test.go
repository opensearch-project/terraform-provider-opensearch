package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchIngestPipeline(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	config := testAccOpensearchIngestPipelineV7

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchIngestPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchIngestPipelineExists("opensearch_ingest_pipeline.test"),
				),
			},
		},
	})
}

func TestAccOpensearchIngestPipeline_importBasic(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	config := testAccOpensearchIngestPipelineV7

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchIngestPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      "opensearch_ingest_pipeline.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchIngestPipelineExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No index template ID is set")
		}

		meta := testAccProvider.Meta()

		var err error
		client, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = client.IngestGetPipeline(rs.Primary.ID).Do(context.TODO())

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchIngestPipelineDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_ingest_pipeline" {
			continue
		}

		meta := testAccProvider.Meta()

		var err error
		client, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = client.IngestGetPipeline(rs.Primary.ID).Do(context.TODO())

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Index template %q still exists", rs.Primary.ID)
	}

	return nil
}

//var testAccOpensearchIngestPipelineV5 = `
//resource "opensearch_ingest_pipeline" "test" {
//  name = "terraform-test"
//  body = <<EOF
//{
//  "description" : "describe pipeline",
//  "processors" : [
//    {
//      "set" : {
//        "field": "foo",
//        "value": "bar"
//      }
//    }
//  ]
//}
//EOF
//}
//`

//var testAccOpensearchIngestPipelineV6 = `
//resource "opensearch_ingest_pipeline" "test" {
//  name = "terraform-test"
//  body = <<EOF
//{
//  "description" : "describe pipeline",
//  "version": 123,
//  "processors" : [
//    {
//      "set" : {
//        "field": "foo",
//        "value": "bar"
//      }
//    }
//  ]
//}
//EOF
//}
//`

var testAccOpensearchIngestPipelineV7 = `
resource "opensearch_ingest_pipeline" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "description" : "describe pipeline",
  "version": 123,
  "processors" : [
    {
      "set" : {
        "field": "foo",
        "value": "bar"
      }
    }
  ]
}
EOF
}
`
