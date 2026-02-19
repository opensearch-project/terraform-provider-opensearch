package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchAnomalyDetection(t *testing.T) {
	randomName := "test-detector" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchAnomalyDetectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchAnomalyDetection(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchAnomalyDetectionExists(fmt.Sprintf("opensearch_anomaly_detection.%s", randomName)),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchAnomalyDetectionUpdate(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchAnomalyDetectionExists(fmt.Sprintf("opensearch_anomaly_detection.%s", randomName)),
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

func testAccOpensearchAnomalyDetection(detectorName string) string {
	return fmt.Sprintf(`
resource "opensearch_audit_config" "test" {
  enabled = true

  audit {
    enable_rest              = true
    disabled_rest_categories = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]

    enable_transport              = true
    disabled_transport_categories = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]

    resolve_bulk_requests = true
    log_request_body      = true
    resolve_indices       = true

    # Note: if set false, AWS OpenSearch will return HTTP 409 (Conflict)
    exclude_sensitive_headers = true

    ignore_users    = ["dashboardserver"]
    ignore_requests = ["SearchRequest", "indices:data/read/*", "/_cluster/health"]
  }

  compliance {
    enabled = true

    # Note: if both internal/external are set true, AWS OpenSearch will return HTTP 409 (Conflict)
    internal_config = true
    external_config = false

    read_metadata_only = true
    read_ignore_users  = ["read-ignore-1"]

    read_watched_field {
      index  = "read-index-1"
      fields = ["field-1", "field-2"]
    }

    read_watched_field {
      index  = "read-index-2"
      fields = ["field-3"]
    }

    write_metadata_only   = true
    write_log_diffs       = false
    write_watched_indices = ["write-index-1", "write-index-2", "log-*", "*"]
    write_ignore_users    = ["write-ignore-1"]
  }
}

resource "opensearch_anomaly_detection" "%[1]s" {
  depends_on = [opensearch_audit_config.test]
  body       = <<EOF
{
  "name": "%[1]s",
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
`, detectorName)
}

func testAccOpensearchAnomalyDetectionUpdate(detectorName string) string {
	return fmt.Sprintf(`
resource "opensearch_audit_config" "test" {
  enabled = true

  audit {
    enable_rest              = true
    disabled_rest_categories = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]

    enable_transport              = true
    disabled_transport_categories = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]

    resolve_bulk_requests = true
    log_request_body      = true
    resolve_indices       = true

    # Note: if set false, AWS OpenSearch will return HTTP 409 (Conflict)
    exclude_sensitive_headers = true

    ignore_users    = ["dashboardserver"]
    ignore_requests = ["SearchRequest", "indices:data/read/*", "/_cluster/health"]
  }

  compliance {
    enabled = true

    # Note: if both internal/external are set true, AWS OpenSearch will return HTTP 409 (Conflict)
    internal_config = true
    external_config = false

    read_metadata_only = true
    read_ignore_users  = ["read-ignore-1"]

    read_watched_field {
      index  = "read-index-1"
      fields = ["field-1", "field-2"]
    }

    read_watched_field {
      index  = "read-index-2"
      fields = ["field-3"]
    }

    write_metadata_only   = true
    write_log_diffs       = false
    write_watched_indices = ["write-index-1", "write-index-2", "log-*", "*"]
    write_ignore_users    = ["write-ignore-1"]
  }
}

resource "opensearch_anomaly_detection" "%[1]s" {
  depends_on = [opensearch_audit_config.test]
  body       = <<EOF
{
  "name": "%[1]s",
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
`, detectorName)
}
