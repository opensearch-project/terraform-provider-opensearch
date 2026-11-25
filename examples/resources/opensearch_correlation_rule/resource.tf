# Basic correlation rule
resource "opensearch_correlation_rule" "multi_stage_attack" {
  name        = "Multi-stage Attack Detection"
  time_window = 300000  # 5 minutes in milliseconds

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
}

# Correlation rule with trigger and actions
resource "opensearch_correlation_rule" "advanced_threat" {
  name        = "Advanced Threat Detection"
  time_window = 600000  # 10 minutes

  correlate {
    index    = "vpc_flow"
    query    = "suspicious_traffic:true"
    category = "network"
  }

  correlate {
    index    = "app_logs"
    query    = "endpoint:/customer_records.txt"
    category = "others_application"
  }

  trigger {
    name     = "Security Alert"
    severity = "1"  # 1 is highest, 5 is lowest

    actions {
      name           = "Send Slack Notification"
      destination_id = opensearch_channel_configuration.security_alerts.id

      subject_template {
        source = "Security Correlation Alert: {{ctx.trigger.name}}"
        lang   = "mustache"
      }

      message_template {
        source = "Correlated findings detected across multiple indices. Details: {{ctx.results}}"
        lang   = "mustache"
      }

      throttle_enabled = true

      throttle {
        unit  = "MINUTES"
        value = 10
      }
    }

    actions {
      name           = "Email Security Team"
      destination_id = opensearch_channel_configuration.email_alerts.id

      subject_template {
        source = "URGENT: Multi-stage Attack Detected"
        lang   = "mustache"
      }

      message_template {
        source = "A multi-stage attack pattern has been detected. Immediate action required."
        lang   = "mustache"
      }

      throttle_enabled = false
    }
  }
}