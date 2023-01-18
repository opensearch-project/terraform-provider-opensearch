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
	config := testAccOpensearchIndexTemplateV7
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
	config := testAccOpensearchIndexTemplateV7

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
		client, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}

		_, err = client.IndexGetIndexTemplate(rs.Primary.ID).Do(context.TODO())

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
		client, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = client.IndexGetIndexTemplate(rs.Primary.ID).Do(context.TODO())

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Index template %q still exists", rs.Primary.ID)
	}

	return nil
}

//
//var testAccOpensearchIndexTemplateV5 = `
//resource "opensearch_index_template" "test" {
//  name = "terraform-test"
//  body = <<EOF
//{
//  "template": "te*",
//  "settings": {
//    "index": {
//      "number_of_shards": 1
//    }
//  },
//  "mappings": {
//    "type1": {
//      "_source": {
//        "enabled": false
//      },
//      "properties": {
//        "host_name": {
//          "type": "keyword"
//        },
//        "created_at": {
//          "type": "date",
//          "format": "EEE MMM dd HH:mm:ss Z YYYY"
//        }
//      }
//    }
//  }
//}
//EOF
//}
//`
//
//var testAccOpensearchIndexTemplateV6 = `
//resource "opensearch_index_template" "test" {
//  name = "terraform-test"
//  body = <<EOF
//{
//  "index_patterns": ["te*", "bar*"],
//  "settings": {
//    "index": {
//      "number_of_shards": 1
//    }
//  },
//  "mappings": {
//    "type1": {
//      "_source": {
//        "enabled": false
//      },
//      "properties": {
//        "host_name": {
//          "type": "keyword"
//        },
//        "created_at": {
//          "type": "date",
//          "format": "EEE MMM dd HH:mm:ss Z YYYY"
//        }
//      }
//    }
//  }
//}
//EOF
//}
//`

var testAccOpensearchIndexTemplateV7 = `
resource "opensearch_index_template" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "index_patterns": ["te*", "bar*"],
  "settings": {
    "index": {
      "number_of_shards": 1
    }
  },
  "mappings": {
    "_source": {
      "enabled": false
    },
    "properties": {
      "host_name": {
        "type": "keyword"
      },
      "created_at": {
        "type": "date",
        "format": "EEE MMM dd HH:mm:ss Z YYYY"
      }
    }
  }
}
EOF
}
`
