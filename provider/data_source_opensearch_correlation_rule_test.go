package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceOpensearchCorrelationRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchCorrelationRuleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceOpensearchCorrelationRuleCheck("data.opensearch_correlation_rule.test"),
				),
			},
		},
	})
}

func testAccDataSourceOpensearchCorrelationRuleCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No correlation rule ID is set")
		}

		// Check that computed attributes are set
		if rs.Primary.Attributes["name"] == "" {
			return fmt.Errorf("Correlation rule name is not set")
		}

		if rs.Primary.Attributes["correlate.#"] == "0" {
			return fmt.Errorf("Correlation rule has no correlate queries")
		}

		return nil
	}
}

const testAccDataSourceOpensearchCorrelationRuleConfig = `
resource "opensearch_correlation_rule" "test" {
  name        = "Test Data Source Rule"
  time_window = 300000

  correlate {
    index    = "vpc_flow"
    query    = "dstaddr:4.5.6.7"
    category = "network"
  }

  correlate {
    index    = "windows"
    query    = "winlog.event_data.SubjectDomainName:TEST*"
    category = "windows"
  }
}

data "opensearch_correlation_rule" "test" {
  rule_id = opensearch_correlation_rule.test.id
}
`
