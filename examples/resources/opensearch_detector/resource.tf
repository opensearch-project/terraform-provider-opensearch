resource "opensearch_detector" "windows_security_detector" {
  name          = "windows-security-monitoring"
  detector_type = "windows"
  enabled       = true

  schedule {
    period {
      interval = 10
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input { # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api
      description = "Monitor Windows security events for potential threats"
      indices     = ["windows-logs"] # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api

      # Use pre-packaged Sigma rules for Windows detection
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }

      pre_packaged_rules {
        id = "847def9e-924d-4e90-b7c4-5f581395a2b4"
      }
    }
  }

  # Configure high-priority trigger for critical events
  triggers {
    name     = "critical-windows-events"
    severity = "1"

    ids = [
      "06724a9a-52fc-11ed-bdc3-0242ac120002",
      "847def9e-924d-4e90-b7c4-5f581395a2b4"
    ]

    sev_levels = ["critical", "high"]
    tags       = ["attack.defense_evasion", "attack.credential_access"]
    
    # detection_types defaults to ["rules"] if not specified

    actions {
      id             = "windows-alert-action"
      name           = "Windows Security Alert"
      destination_id = "${opensearch_channel_configuration.security_notifications.id}"

      subject_template {
        source = "🚨 Windows Security Alert: {{ctx.detector.name}}"
        lang   = "mustache"
      }

      message_template {
        source = <<-EOT
Detector: {{ctx.detector.name}}
Severity: {{ctx.trigger.severity}} (Critical)
Time: {{ctx.periodStart}} to {{ctx.periodEnd}}

Description: Potential security threat detected in Windows logs.
Please investigate immediately.

Triggered Rules:
{{#ctx.results}}
- Rule ID: {{_source.rule_id}}
- Event: {{_source.event_type}}
{{/ctx.results}}
EOT
        lang   = "mustache"
      }

      throttle_enabled = true

      throttle {
        unit  = "MINUTES"
        value = 15
      }
    }
  }
}

# Example with custom rules and multiple triggers
resource "opensearch_detector" "advanced_linux_detector" {
  name          = "advanced-linux-security"
  detector_type = "linux"
  enabled       = true

  schedule {
    period {
      interval = 5
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input { # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api
      description = "Advanced Linux security monitoring with custom rules"
      indices     = ["linux-logs"] # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api

      # Custom rules (UUIDs would be real custom rule IDs)
      custom_rules {
        id = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
      }

      custom_rules {
        id = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
      }

      # Also include some pre-packaged rules
      pre_packaged_rules {
        id = "73a883d0-0348-4be4-a8d8-51031c2564f8"
      }
    }
  }

  # High severity trigger
  triggers {
    name     = "high-severity-linux-events"
    severity = "1"

    ids = [
      "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "73a883d0-0348-4be4-a8d8-51031c2564f8"
    ]

    sev_levels = ["critical", "high"]
    tags       = ["attack.persistence", "attack.privilege_escalation"]

    actions {
      id             = "high-severity-action"
      name           = "High Severity Linux Alert"
      destination_id = "${opensearch_channel_configuration.security_team.id}"

      subject_template {
        source = "🚨 HIGH SEVERITY: {{ctx.detector.name}}"
        lang   = "mustache"
      }

      message_template {
        source = "High severity security event detected on Linux systems. Immediate investigation required."
        lang   = "mustache"
      }

      throttle_enabled = true
      throttle {
        unit  = "MINUTES"
        value = 10
      }
    }
  }

  # Medium severity trigger
  triggers {
    name     = "medium-severity-linux-events"
    severity = "2"

    ids = [
      "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
    ]

    sev_levels = ["medium"]
    tags       = ["attack.discovery"]

    actions {
      id             = "medium-severity-action"
      name           = "Medium Severity Linux Alert"
      destination_id = "${opensearch_channel_configuration.monitoring_team.id}"

      subject_template {
        source = "⚠️ Medium Severity: {{ctx.detector.name}}"
        lang   = "mustache"
      }

      message_template {
        source = "Medium severity security event detected. Please review when convenient."
        lang   = "mustache"
      }

      throttle_enabled = true
      throttle {
        unit  = "HOURS"
        value = 1
      }
    }
  }
}

# Simple network detector (demonstrates default detection_types behavior)
resource "opensearch_detector" "network_intrusion_detector" {
  name          = "network-intrusion-detection"
  detector_type = "network"
  enabled       = true

  schedule {
    period {
      interval = 2
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input { # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api
      description = "Detect network intrusions and suspicious traffic"
      indices     = ["network-logs"] # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api

      pre_packaged_rules {
        id = "1a4bd6e3-4c6e-405d-a9a3-53a116e341d4"
      }
    }
  }

  triggers {
    name     = "network-intrusion-alert"
    severity = "2"
    
    ids = ["1a4bd6e3-4c6e-405d-a9a3-53a116e341d4"]
    sev_levels = ["high", "medium"]
    
    # detection_types will automatically default to ["rules"] since it's not specified
    
    actions {
      id             = "network-alert-action"
      name           = "Network Intrusion Alert"
      destination_id = "network-security-channel"
      
      subject_template {
        source = "Network Intrusion Detected"
        lang   = "mustache"
      }
      
      message_template {
        source = "Network intrusion detected by {{ctx.detector.name}}"
        lang   = "mustache"
      }
    }
  }
}

# CloudTrail detector for AWS monitoring
resource "opensearch_detector" "cloudtrail_detector" {
  name          = "aws-cloudtrail-monitoring"
  detector_type = "cloudtrail"
  enabled       = true

  schedule {
    period {
      interval = 1
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input { # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api
      description = "Monitor AWS CloudTrail for suspicious activities"
      indices     = ["cloudtrail-logs"] # only one is supported right now https://docs.opensearch.org/latest/security-analytics/api-tools/detector-api/#create-detector-api

      pre_packaged_rules {
        id = "aws-cloudtrail-rule-1"
      }

      pre_packaged_rules {
        id = "aws-cloudtrail-rule-2"
      }
    }
  }

  triggers {
    name     = "aws-security-events"
    severity = "2"

    ids = [
      "aws-cloudtrail-rule-1",
      "aws-cloudtrail-rule-2"
    ]

    sev_levels = ["high", "medium"]
    tags       = ["attack.initial_access", "attack.persistence"]

    actions {
      id             = "aws-security-action"
      name           = "AWS Security Alert"
      destination_id = "${opensearch_channel_configuration.aws_security_team.id}"

      subject_template {
        source = "AWS Security Event: {{ctx.detector.name}}"
        lang   = "mustache"
      }

      message_template {
        source = "Suspicious AWS activity detected in CloudTrail logs."
        lang   = "mustache"
      }

      throttle_enabled = false
    }
  }
}