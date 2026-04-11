package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func TestAccOpensearchIndexTemplate(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var config string = testAccOpensearchIndexTemplateV7
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchIndexTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchIndexTemplateExists("opensearch_index_template.test"),
				),
			},
			{
				Config: testAccOpensearchIndexTemplateV7Updated,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchIndexTemplateExists("opensearch_index_template.test"),
				),
			},
		},
	})
}

func TestAccOpensearchIndexTemplate_importBasic(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var config string = testAccOpensearchIndexTemplateV7

	resource.Test(t, resource.TestCase{

		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchIndexTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      "opensearch_index_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchIndexTemplateExists(name string) resource.TestCheckFunc {
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
		client, err := getOpenSearchClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = client.Client.Template.Get(context.TODO(), &opensearchapi.TemplateGetReq{
			Templates: []string{rs.Primary.ID},
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchIndexTemplateDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_index_template" {
			continue
		}

		meta := testAccProvider.Meta()

		var err error
		client, err := getOpenSearchClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = client.Client.Template.Get(context.TODO(), &opensearchapi.TemplateGetReq{
			Templates: []string{rs.Primary.ID},
		})

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Index template %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchIndexTemplateV7 = `
resource "opensearch_index_template" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "index_patterns": [
    "logs-2020-01-*"
  ],
  "aliases": {
    "my_logs": {}
  },
  "mappings": {
    "properties": {
      "timestamp": {
        "type": "date",
        "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis"
      },
      "value": {
        "type": "double"
      }
    }
  }
}
EOF
}
`

var testAccOpensearchIndexTemplateV7Updated = `
resource "opensearch_index_template" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "index_patterns": [
    "logs-2020-01-*",
    "logs-2020-02-*"
  ],
  "aliases": {
    "my_logs": {}
  },
  "mappings": {
    "properties": {
      "timestamp": {
        "type": "date",
        "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis"
      },
      "value": {
        "type": "double"
      }
    }
  }
}
EOF
}
`
