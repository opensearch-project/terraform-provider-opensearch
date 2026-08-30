# Integration Test: Security Monitoring & Alerting
# Tests security audit log monitoring with anomaly detection:
# 1. Anomaly detection (unusual access patterns — requires indexed documents)
# 2. Notification channel (Slack)
# 3. Monitor (failed login threshold)
# 4. Security analyst role
# 5. Analyst user account
# 6. Roles mapping

provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

# ---------------------------------------------------------------------------
# 1. Anomaly detection detector — requires index data (pre-populated)
# ---------------------------------------------------------------------------
run "create_anomaly_detector" {
  module {
    source = "./modules/opensearch_anomaly_detection"
  }

  variables {
    name = "integration-security-detector"
    body = jsonencode({
      name        = "Security Audit Anomaly Detector"
      description = "Detects unusual access patterns in audit logs"
      time_field  = "@timestamp"
      indices     = ["security-audit-logs"]
      detection_interval = {
        period = {
          interval = 10
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
          feature_name    = "doc_count"
          feature_enabled = true
          importance      = 1
          aggregation_query = {
            doc_count = {
              value_count = {
                field = "source_ip"
              }
            }
          }
        }
      ]
    })
  }

  assert {
    condition     = opensearch_anomaly_detection.this.id != ""
    error_message = "Anomaly detector should be created"
  }
}

# ---------------------------------------------------------------------------
# 2. Notification channel (Slack)
# ---------------------------------------------------------------------------
run "create_channel" {
  module {
    source = "./modules/opensearch_channel_configuration"
  }

  variables {
    body = jsonencode({
      config = {
        name        = "integration-security-channel"
        description = "Security alerts channel"
        config_type = "slack"
        is_enabled  = true
        slack = {
          url = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
        }
      }
    })
  }

  assert {
    condition     = opensearch_channel_configuration.this.id != ""
    error_message = "Channel configuration should be created"
  }
}

# ---------------------------------------------------------------------------
# 3. Monitor — failed login threshold alert
# ---------------------------------------------------------------------------
run "create_monitor" {
  module {
    source = "./modules/opensearch_monitor"
  }

  variables {
    name = "integration-failed-login-monitor"
    body = jsonencode({
      type         = "monitor"
      name         = "Failed Login Threshold Monitor"
      monitor_type = "query_level_monitor"
      enabled      = true
      schedule = {
        period = {
          interval = 1
          unit     = "Minutes"
        }
      }
      inputs = [
        {
          search = {
            indices = ["security-audit-logs"]
            query = {
              size = 0
              query = {
                bool = {
                  filter = [
                    {
                      range = {
                        "@timestamp" = {
                          from          = "{{period_end}}||-1m"
                          to            = "{{period_end}}"
                          include_lower = true
                          include_upper = true
                        }
                      }
                    },
                    {
                      term = {
                        result = "failure"
                      }
                    }
                  ]
                }
              }
              aggregations = {
                failed_count = {
                  value_count = {
                    field = "_index"
                  }
                }
              }
            }
          }
        }
      ]
      triggers = [
        {
          name     = "Failed Login Alert"
          severity = "1"
          condition = {
            script = {
              source = "ctx.results[0].aggregations.failed_count.value \u003e 5"
              lang   = "painless"
            }
          }
          actions = [
            {
              name           = "notify-slack"
              destination_id = run.create_channel.id
              message_template = {
                source = "Alert: More than 5 failed logins detected in the last minute."
                lang   = "mustache"
              }
              throttle_enabled = true
              throttle = {
                value = 10
                unit  = "MINUTES"
              }
            }
          ]
        }
      ]
    })
  }

  assert {
    condition     = opensearch_monitor.this.id != ""
    error_message = "Monitor should be created"
  }
}

# ---------------------------------------------------------------------------
# 4. Security analyst role
# ---------------------------------------------------------------------------
run "create_analyst_role" {
  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name   = "integration-security-analyst"
    description = "Security analyst with access to audit logs and anomaly detection"
    cluster_permissions = [
      "cluster:monitor/health",
      "cluster:monitor/state",
      "cluster:admin/opendistro/alerting/monitor/search"
    ]
    index_permissions = [
      {
        index_patterns          = ["security-audit-logs"]
        allowed_actions         = ["read", "indices:monitor/stats"]
        document_level_security = ""
        field_level_security    = []
        masked_fields           = []
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "integration-security-analyst"
    error_message = "Analyst role name should match"
  }
}

# ---------------------------------------------------------------------------
# 5. Analyst user account
# ---------------------------------------------------------------------------
run "create_analyst_user" {
  module {
    source = "./modules/opensearch_user"
  }

  variables {
    username    = "integration-analyst"
    password    = "AnalystP@ssw0rd123!"
    description = "Integration test security analyst"
    attributes = {
      department = "security"
    }
  }

  assert {
    condition     = opensearch_user.this.username == "integration-analyst"
    error_message = "Analyst username should match"
  }
}

# ---------------------------------------------------------------------------
# 6. Roles mapping — analyst user to analyst role
# ---------------------------------------------------------------------------
run "create_roles_mapping" {
  module {
    source = "./modules/opensearch_roles_mapping"
  }

  variables {
    role_name = run.create_analyst_role.role_name
    users     = ["integration-analyst"]
  }

  assert {
    condition     = opensearch_roles_mapping.this.role_name == "integration-security-analyst"
    error_message = "Roles mapping should reference the analyst role"
  }
}
