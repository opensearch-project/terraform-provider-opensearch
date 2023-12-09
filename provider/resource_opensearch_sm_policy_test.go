package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchSMPolicy(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool = true

	config := testAccOpensearchSMPolicyV7

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)

			if !allowed {
				t.Skip("OpenSearch Snapshot Management only supported on Opensearch >= 2.1")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchSMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchSMPolicyExists("opensearch_sm_policy.test_policy"),
					resource.TestCheckResourceAttr(
						"opensearch_sm_policy.test_policy",
						"policy_name",
						"test_policy",
					),
				),
			},
		},
	})
}

func testCheckOpensearchSMPolicyExists(name string) resource.TestCheckFunc {
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
		_, err = resourceOpensearchGetSMPolicy(rs.Primary.Attributes["policy_name"], meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchSMPolicyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_sm_policy" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		if err != nil {
			return err
		}
		_, err = resourceOpensearchGetSMPolicy(rs.Primary.Attributes["policy_name"], meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("OpenDistroSMPolicy %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchSMPolicyV7 = `
resource "opensearch_snapshot_repository" "test" {
  name = "terraform-test"
  type = "fs"

  settings = {
    location = "/tmp/opensearch"
  }
}

resource "opensearch_sm_policy" "test_policy" {
  policy_name = "test_policy"
  body        = <<EOF
  {
		"enabled": true,
		"description": "Test policy",
		"creation": {
			"schedule": {
				"cron": {
					"expression": "0 0 * * *",
					"timezone": "UTC"
				}
			},
			"time_limit": "1h"
		},
		"deletion": {
			"schedule": {
				"cron": {
					"expression": "0 1 * * *",
					"timezone": "UTC"
				}
			},
			"condition": {
				"max_age": "14d",
				"max_count": 400,
				"min_count": 1
			},
			"time_limit": "1h"
		},
		"snapshot_config": {
			"timezone": "UTC",
			"indices": "*",
			"repository": "${opensearch_snapshot_repository.test.name}"
		}
	}
  EOF
}
`
