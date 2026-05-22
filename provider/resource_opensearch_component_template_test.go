package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func TestAccOpensearchComponentTemplate(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool = true
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("/_component_template endpoint only supported on OS >= 2.0.0")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchComponentTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchComponentTemplate,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchComponentTemplateExists("opensearch_component_template.test"),
				),
			},
			{
				Config: testAccOpensearchComponentTemplateUpdated,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchComponentTemplateExists("opensearch_component_template.test"),
				),
			},
		},
	})
}

func TestAccOpensearchComponentTemplate_importBasic(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool = true
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("/_component_template endpoint only supported on OS >= 2.0.0")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchComponentTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchComponentTemplate,
			},
			{
				ResourceName:      "opensearch_component_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchComponentTemplateExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No component template ID is set")
		}

		meta := testAccProvider.Meta()

		client, err := getOpenSearchClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}

		_, err = client.Client.ComponentTemplate.Get(context.TODO(), &opensearchapi.ComponentTemplateGetReq{
			ComponentTemplate: rs.Primary.ID,
		})

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchComponentTemplateDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_component_template" {
			continue
		}

		meta := testAccProvider.Meta()

		client, err := getOpenSearchClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}

		_, err = client.Client.ComponentTemplate.Get(context.TODO(), &opensearchapi.ComponentTemplateGetReq{
			ComponentTemplate: rs.Primary.ID,
		})

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Component template %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchComponentTemplate = `
resource "opensearch_component_template" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "template": {
    "settings": {
      "index": {
        "number_of_shards": 1
      }
    },
    "mappings": {
      "properties": {
        "host_name": {
          "type": "keyword"
        },
        "created_at": {
          "type": "date",
          "format": "EEE MMM dd HH:mm:ss Z yyyy"
        }
      }
    },
    "aliases": {
      "mydata": { }
    }
  }
}
EOF
}
`

var testAccOpensearchComponentTemplateUpdated = `
resource "opensearch_component_template" "test" {
  name = "terraform-test"
  body = <<EOF
{
  "template": {
    "settings": {
      "index": {
        "number_of_shards": 2
      }
    },
    "mappings": {
      "properties": {
        "host_name": {
          "type": "keyword"
        },
        "created_at": {
          "type": "date",
          "format": "EEE MMM dd HH:mm:ss Z yyyy"
        }
      }
    },
    "aliases": {
      "mydata": { }
    }
  }
}
EOF
}
`
