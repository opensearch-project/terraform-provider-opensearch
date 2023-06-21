package provider

import (
	"context"
	"fmt"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroISMPolicy(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	meta := provider.Meta()
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		t.Skipf("err: %s", err)
	}
	var allowed bool

	var config string
	switch esClient.(type) {
	case *elastic6.Client:
		allowed = true
		config = testAccOpensearchOpenDistroISMPolicyV6
	default:
		allowed = true
		config = testAccOpensearchOpenDistroISMPolicyV7
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("OpenDistroISMPolicies only supported on ES 6.")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchISMPolicyExists("opensearch_ism_policy.test_policy"),
					resource.TestCheckResourceAttr(
						"opensearch_ism_policy.test_policy",
						"policy_id",
						"test_policy",
					),
				),
			},
		},
	})
}

func testCheckOpensearchISMPolicyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch esClient.(type) {
		case *elastic7.Client:
			_, err = resourceOpensearchGetOpenDistroISMPolicy(rs.Primary.ID, meta.(*ProviderConf))
		case *elastic6.Client:
			_, err = resourceOpensearchGetOpenDistroISMPolicy(rs.Primary.ID, meta.(*ProviderConf))
		default:
		}

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchISMPolicyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_ism_policy" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch esClient.(type) {
		case *elastic7.Client:
			_, err = resourceOpensearchGetOpenDistroISMPolicy(rs.Primary.ID, meta.(*ProviderConf))
		case *elastic6.Client:
			_, err = resourceOpensearchGetOpenDistroISMPolicy(rs.Primary.ID, meta.(*ProviderConf))
		default:
		}

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("OpenDistroISMPolicy %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchOpenDistroISMPolicyV6 = `
resource "opensearch_ism_policy" "test_policy" {
  policy_id = "test_policy"
  body      = <<EOF
  {
		"policy": {
		  "description": "ingesting logs",
		  "default_state": "ingest",
		  "error_notification": {
        "destination": {
          "slack": {
            "url": "https://webhook.slack.example.com"
          }
        },
        "message_template": {
          "lang": "mustache",
          "source": "The index *{{ctx.index}}* failed to rollover."
        }
      },
		  "states": [
				{
				  "name": "ingest",
				  "actions": [{
					  "rollover": {
						"min_doc_count": 5
					  }
					}],
				  "transitions": [{
					  "state_name": "search"
					}]
				},
				{
				  "name": "search",
				  "actions": [],
				  "transitions": [{
					  "state_name": "delete",
					  "conditions": {
						"min_index_age": "5m"
					  }
					}]
				},
				{
				  "name": "delete",
				  "actions": [{
					  "delete": {}
					}],
				  "transitions": []
				}
			]
		}
	}
  EOF
}
`

var testAccOpensearchOpenDistroISMPolicyV7 = `
resource "opensearch_ism_policy" "test_policy" {
  policy_id = "test_policy"
  body      = <<EOF
  {
		"policy": {
		  "description": "ingesting logs",
		  "default_state": "ingest",
      "ism_template": [{
        "index_patterns": ["foo-*"],
        "priority": 0
			}],
		  "error_notification": {
        "destination": {
          "slack": {
            "url": "https://webhook.slack.example.com"
          }
        },
        "message_template": {
          "lang": "mustache",
          "source": "The index *{{ctx.index}}* failed to rollover."
        }
      },
		  "states": [
				{
				  "name": "ingest",
				  "actions": [{
					  "rollover": {
						  "min_doc_count": 5
					  },
	          "retry": {
	            "backoff": "exponential",
	            "count": 3,
	            "delay": "1m"
	          }
					}],
				  "transitions": [{
					  "state_name": "search"
					}]
				},
				{
				  "name": "search",
				  "actions": [],
				  "transitions": [{
					  "state_name": "delete",
					  "conditions": {
						"min_index_age": "5m"
					  }
					}]
				},
				{
				  "name": "delete",
				  "actions": [{
					  "delete": {},
	          "retry": {
	            "backoff": "exponential",
	            "count": 3,
	            "delay": "1m"
	          }
					}],
				  "transitions": []
				}
			]
		}
	}
  EOF
}
`
