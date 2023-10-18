package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchAnomalyDetection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchAnomalyDetectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchAnomalyDetection,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchAnomalyDetectionExists("opensearch_anomaly_detection.test-detector12"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchAnomalyDetectionUpdate,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchAnomalyDetectionExists("opensearch_anomaly_detection.test-detector12"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckOpensearchAnomalyDetectionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No monitor ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchAnomalyDetectionGet(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchAnomalyDetectionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_anomaly_detection" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchAnomalyDetectionGet(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Monitor %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchAnomalyDetection = `
resource "opensearch_anomaly_detection" "test-detector12" {
  body = <<EOF
{
  "name": "test-detector12",
  "description": "Test detector",
  "time_field": "@timestamp",
  "indices": [
    "security-auditlog*"
  ],
  "feature_attributes": [
    {
      "feature_name": "test",
      "feature_enabled": true,
      "aggregation_query": {
        "test": {
          "value_count": {
            "field": "audit_category.keyword"
          }
        }
      }
    }
  ],
  "filter_query": {
    "bool": {
      "filter": [
        {
          "range": {
            "value": {
              "gt": 1
            }
          }
        }
      ],
      "adjust_pure_negative": true,
      "boost": 1
    }
  },
  "detection_interval": {
    "period": {
      "interval": 1,
      "unit": "Minutes"
    }
  },
  "window_delay": {
    "period": {
      "interval": 1,
      "unit": "Minutes"
    }
  },
  "result_index" : "opensearch-ad-plugin-result-test"
}
EOF
}
`

var testAccOpensearchAnomalyDetectionUpdate = `
resource "opensearch_anomaly_detection" "test-detector12" {
  body = <<EOF
{
  "name": "test-detector12",
  "description": "Test detector 12",
  "time_field": "@timestamp",
  "indices": [
    "security-auditlog*"
  ],
  "feature_attributes": [
    {
      "feature_name": "test",
      "feature_enabled": true,
      "aggregation_query": {
        "test": {
          "value_count": {
            "field": "audit_category.keyword"
          }
        }
      }
    }
  ],
  "filter_query": {
    "bool": {
      "filter": [
        {
          "range": {
            "value": {
              "gt": 1
            }
          }
        }
      ],
      "adjust_pure_negative": true,
      "boost": 1
    }
  },
  "detection_interval": {
    "period": {
      "interval": 1,
      "unit": "Minutes"
    }
  },
  "window_delay": {
    "period": {
      "interval": 1,
      "unit": "Minutes"
    }
  },
  "result_index" : "opensearch-ad-plugin-result-test"
}
EOF
}
`
