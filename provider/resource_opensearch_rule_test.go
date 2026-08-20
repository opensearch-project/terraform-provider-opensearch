package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testCheckOpensearchRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchRuleConfig(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchRuleExists("opensearch_rule.test"),
					resource.TestCheckResourceAttr("opensearch_rule.test", "category", "windows"),
					resource.TestCheckResourceAttrSet("opensearch_rule.test", "rule_id"),
					resource.TestCheckResourceAttrSet("opensearch_rule.test", "version"),
				),
			},
			{
				Config: testAccOpensearchRuleConfigUpdated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchRuleExists("opensearch_rule.test"),
					resource.TestCheckResourceAttr("opensearch_rule.test", "category", "windows"),
				),
			},
		},
	})
}

func testCheckOpensearchRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no rule ID is set")
		}

		meta := testAccProvider.Meta()
		_, err := resourceOpensearchGetRule(rs.Primary.ID, meta)
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchRuleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_rule" {
			continue
		}

		meta := testAccProvider.Meta()
		_, err := resourceOpensearchGetRule(rs.Primary.ID, meta)
		if err == nil {
			return fmt.Errorf("rule still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccOpensearchRuleConfig() string {
	return `
resource "opensearch_rule" "test" {
  category = "windows"
  rule     = <<EOF
title: Test Rule
id: 12345678-1234-1234-1234-123456789012
description: A test rule for Terraform
status: experimental
author: Terraform Test
date: 2024/01/01
logsource:
    product: windows
    service: system
detection:
    selection:
        EventID: 4624
    condition: selection
level: medium
falsepositives:
    - None
EOF
}
`
}

func testAccOpensearchRuleConfigUpdated() string {
	return `
resource "opensearch_rule" "test" {
  category = "windows"
  forced   = true
  rule     = <<EOF
title: Test Rule Updated
id: 12345678-1234-1234-1234-123456789012
description: An updated test rule for Terraform
status: stable
author: Terraform Test
date: 2024/01/01
modified: 2024/01/02
logsource:
    product: windows
    service: system
detection:
    selection:
        EventID: 4624
    condition: selection
level: high
falsepositives:
    - None
EOF
}
`
}
