package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccOpensearchDataSourceRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDataSourceRuleConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_rule.test", "rules.#"),
				),
			},
		},
	})
}

func TestAccOpensearchDataSourceRuleWithFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDataSourceRuleConfigWithFilters(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_rule.filtered", "rules.#"),
				),
			},
		},
	})
}

func testAccOpensearchDataSourceRuleConfig() string {
	return `
data "opensearch_rule" "test" {
  category = "windows"
}
`
}

func testAccOpensearchDataSourceRuleConfigWithFilters() string {
	return fmt.Sprintf(`
resource "opensearch_rule" "test_rule" {
  category = "windows"
  rule     = <<EOF
title: Data Source Test Rule
id: 87654321-4321-4321-4321-210987654321
description: A test rule for data source
status: experimental
author: Terraform Test
date: 2024/01/01
logsource:
    product: windows
    service: security
detection:
    selection:
        EventID: 4625
    condition: selection
level: medium
falsepositives:
    - None
EOF
}

data "opensearch_rule" "filtered" {
  category = "windows"
  level    = "medium"
  status   = "experimental"
  
  depends_on = [opensearch_rule.test_rule]
}
`)
}
