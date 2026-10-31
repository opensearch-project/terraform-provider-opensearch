provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

# Plan-only test: Anomaly detection requires indexed documents in the target
# index before a detector can be created, which exceeds Terraform's scope.
run "validate_detector_config" {
  command   = plan
  state_key = "validate_detector_config"

  module {
    source = "./modules/opensearch_anomaly_detection"
  }

  variables {
    name = "test-detector"
    body = jsonencode({
      name        = "Test Detector"
      description = "Test anomaly detector"
      time_field  = "@timestamp"
      indices     = ["logs-test"]
      detection_interval = {
        period = {
          interval = 10
          unit     = "Minutes"
        }
      }
      feature_attributes = []
    })
  }
}

run "validate_detector_full_config" {
  command   = plan
  state_key = "validate_detector_full_config"

  module {
    source = "./modules/opensearch_anomaly_detection"
  }

  variables {
    name = "test-detector-full"
    body = jsonencode({
      name        = "Full Test Detector"
      description = "Full anomaly detector with all fields"
      time_field  = "@timestamp"
      indices     = ["logs-test"]
      detection_interval = {
        period = {
          interval = 5
          unit     = "Minutes"
        }
      }
      window_delay = {
        period = {
          interval = 1
          unit     = "Minutes"
        }
      }
      feature_attributes = [
        {
          feature_name    = "value_count"
          feature_enabled = true
          importance      = 1
          aggregation_query = {
            value_count = {
              value = {
                field = "value"
              }
            }
          }
        },
        {
          feature_name    = "avg_value"
          feature_enabled = true
          importance      = 2
          aggregation_query = {
            avg_value = {
              avg = {
                field = "value"
              }
            }
          }
        }
      ]
      category_field = ["category"]
      ui_metadata = {
        index = "logs-test"
      }
    })
  }
}
