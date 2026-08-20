package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchCorrelationRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchCorrelationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchCorrelationRuleBasic,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCorrelationRuleExists("opensearch_correlation_rule.test"),
					resource.TestCheckResourceAttr("opensearch_correlation_rule.test", "name", "Test Correlation Rule"),
					resource.TestCheckResourceAttr("opensearch_correlation_rule.test", "correlate.#", "2"),
					resource.TestCheckResourceAttr("opensearch_correlation_rule.test", "time_window", "300000"),
				),
			},
			{
				Config: testAccOpensearchCorrelationRuleUpdated,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCorrelationRuleExists("opensearch_correlation_rule.test"),
					resource.TestCheckResourceAttr("opensearch_correlation_rule.test", "name", "Updated Correlation Rule"),
					resource.TestCheckResourceAttr("opensearch_correlation_rule.test", "correlate.#", "3"),
				),
			},
		},
	})
}

func testCheckOpensearchCorrelationRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No correlation rule ID is set")
		}

		meta := testAccProvider.Meta()
		_, err := resourceOpensearchGetCorrelationRule(rs.Primary.ID, meta)
		return err
	}
}

func testCheckOpensearchCorrelationRuleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_correlation_rule" {
			continue
		}

		meta := testAccProvider.Meta()
		_, err := resourceOpensearchGetCorrelationRule(rs.Primary.ID, meta)
		if err == nil {
			return fmt.Errorf("Correlation rule still exists")
		}
	}

	return nil
}

const testAccOpensearchCorrelationRuleBasic = `
resource "opensearch_correlation_rule" "test" {
  name        = "Test Correlation Rule"
  time_window = 300000

  correlate {
    index    = "vpc_flow"
    query    = "dstaddr:4.5.6.7 or dstaddr:4.5.6.6"
    category = "network"
    field    = "source.ip"
  }

  correlate {
    index    = "windows"
    query    = "winlog.event_data.SubjectDomainName:NTAUTHORI*"
    category = "windows"
    field    = "user.id"
  }
}
`

const testAccOpensearchCorrelationRuleUpdated = `
resource "opensearch_correlation_rule" "test" {
  name        = "Updated Correlation Rule"
  time_window = 600000

  correlate {
    index    = "vpc_flow"
    query    = "dstaddr:4.5.6.7 or dstaddr:4.5.6.6"
    category = "network"
    field    = "source.ip"
  }

  correlate {
    index    = "windows"
    query    = "winlog.event_data.SubjectDomainName:NTAUTHORI*"
    category = "windows"
    field    = "user.id"
  }

  correlate {
    index    = "ad_logs"
    query    = "ResultType:50126"
    category = "ad_ldap"
    field    = "user.id"
  }

  trigger {
    name     = "Test Trigger"
    severity = "1"

    actions {
      name           = "Test Action"
      destination_id = "test-destination-id"

      subject_template {
        source = "Correlation Alert: {{ctx.trigger.name}}"
        lang   = "mustache"
      }

      message_template {
        source = "Correlated findings detected"
        lang   = "mustache"
      }

      throttle_enabled = true

      throttle {
        unit  = "MINUTES"
        value = 10
      }
    }
  }
}
`
