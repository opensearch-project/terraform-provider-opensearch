package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = osClient.IndexGetIndexTemplate(rs.Primary.ID).Do(context.TODO())

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
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = osClient.IndexGetIndexTemplate(rs.Primary.ID).Do(context.TODO())

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
	"template": {
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
  }
EOF
}
`
