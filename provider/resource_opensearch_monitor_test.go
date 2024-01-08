package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroMonitor(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMonitorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchOpenDistroMonitor,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMonitorExists("opensearch_monitor.test_monitor"),
				),
			},
		},
	})
}

func testCheckOpensearchMonitorExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No monitor ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		resp, err := resourceOpensearchOpenDistroGetMonitor(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		respMonitor := resp.Monitor
		normalizeMonitor(respMonitor)
		monitorJson, err := json.Marshal(respMonitor)

		if err != nil {
			return err
		}

		originalMonitorJsonNormalized, err := structure.NormalizeJsonString(testAccOpensearchOpenDistroMonitorJSON)
		if err != nil {
			return err
		}
		monitorJsonNormalized, err := structure.NormalizeJsonString(string(monitorJson))
		if err != nil {
			return err
		}

		diff := diffSuppressMonitor("", monitorJsonNormalized, originalMonitorJsonNormalized, nil)
		if !diff {

			return fmt.Errorf("Monitor does not match.\nOld monitor: %s,\nNew monitor: %s", originalMonitorJsonNormalized, monitorJsonNormalized)
		}

		return nil
	}
}

func testCheckOpensearchMonitorDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_monitor" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchOpenDistroGetMonitor(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Monitor %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchOpenDistroMonitorJSON = `
{
	"name": "test-monitor",
	"type": "monitor",
	"monitor_type": "query_level_monitor",
	"owner": "alerting",
	"enabled": true,
	"schedule": {
		"period": {
		"interval": 1,
		"unit": "MINUTES"
		}
	},
	"inputs": [
		{
		"search": {
			"indices": ["*"],
			"query": {
			"size": 0,
			"query": {
				"bool": {
				"adjust_pure_negative": true,
				"boost": 1,
				"filter": [
					{
					"range": {
						"@timestamp": {
						"boost": 1,
						"from": "||-1h",
						"to": "",
						"include_lower": true,
						"include_upper": true,
						"format": "epoch_millis"
						}
					}
					}
				]
				}
			}
			}
		}
		}
	],
	"triggers": []
}
`

var testAccOpensearchOpenDistroMonitor = `
resource "opensearch_monitor" "test_monitor" {
  body = <<EOF
  {
	"name": "test-monitor",
	"type": "monitor",
	"monitor_type": "query_level_monitor",
	"owner": "alerting",
	"enabled": true,
	"schedule": {
		"period": {
		"interval": 1,
		"unit": "MINUTES"
		}
	},
	"inputs": [
		{
		"search": {
			"indices": ["*"],
			"query": {
			"size": 0,
			"query": {
				"bool": {
				"adjust_pure_negative": true,
				"boost": 1,
				"filter": [
					{
					"range": {
						"@timestamp": {
						"boost": 1,
						"from": "||-1h",
						"to": "",
						"include_lower": true,
						"include_upper": true,
						"format": "epoch_millis"
						}
					}
					}
				]
				}
			}
			}
		}
		}
	],
	"triggers": []
}
EOF
}
`
