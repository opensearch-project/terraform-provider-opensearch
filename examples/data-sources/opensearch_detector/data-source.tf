# Get detector by ID
data "opensearch_detector" "security_detector_by_id" {
  detector_id = "dc2VB4QBrbtylUb_Hfa3"
}

# Output detector information
output "detector_by_id_info" {
  value = {
    name          = data.opensearch_detector.security_detector_by_id.name
    type          = data.opensearch_detector.security_detector_by_id.detector_type
    enabled       = data.opensearch_detector.security_detector_by_id.enabled
    last_updated  = data.opensearch_detector.security_detector_by_id.last_update_time
    schedule      = data.opensearch_detector.security_detector_by_id.schedule[0].period[0]
  }
}

# Get detector by name
data "opensearch_detector" "windows_detector_by_name" {
  name = "windows-security-monitoring"
}

# Use detector information to create related resources
resource "opensearch_channel_configuration" "detector_notifications" {
  name = "notifications-for-${data.opensearch_detector.windows_detector_by_name.name}"
  
  config_type = "slack"
  description = "Slack notifications for ${data.opensearch_detector.windows_detector_by_name.name} detector alerts"
  
  slack {
    url = "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
  }
}

# Get first detector of a specific type
data "opensearch_detector" "linux_detector_by_type" {
  detector_type = "linux"
}

# Output linux detector configuration
output "linux_detector_config" {
  value = {
    id            = data.opensearch_detector.linux_detector_by_type.id
    name          = data.opensearch_detector.linux_detector_by_type.name
    enabled       = data.opensearch_detector.linux_detector_by_type.enabled
    inputs        = data.opensearch_detector.linux_detector_by_type.inputs
    triggers      = data.opensearch_detector.linux_detector_by_type.triggers
  }
}

# Example: Use detector data source with local resources
resource "opensearch_detector" "example_detector" {
  name          = "example-windows-detector"
  detector_type = "windows" 
  enabled       = true

  schedule {
    period {
      interval = 15
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Example Windows detector"
      indices     = ["windows-events"]
      
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}

# Get the detector we just created
data "opensearch_detector" "example_detector_data" {
  detector_id = opensearch_detector.example_detector.id
}

# Create dashboard object based on detector configuration
resource "opensearch_dashboard_object" "detector_dashboard" {
  body = jsonencode({
    version = "1.0.0"
    objects = [{
      id         = "detector-${data.opensearch_detector.example_detector_data.id}-dashboard"
      type       = "dashboard" 
      attributes = {
        title       = "Security Dashboard for ${data.opensearch_detector.example_detector_data.name}"
        description = "Dashboard showing alerts and findings for detector: ${data.opensearch_detector.example_detector_data.name}"
        type        = data.opensearch_detector.example_detector_data.detector_type
        # ... other dashboard configuration
      }
    }]
  })
}

# Example: Conditional resource creation based on detector configuration
locals {
  # Check if detector has high severity triggers
  has_critical_triggers = length([
    for trigger in data.opensearch_detector.linux_detector_by_type.triggers : trigger
    if trigger.severity == "1"
  ]) > 0
}

# Create escalation channel only if detector has critical triggers
resource "opensearch_channel_configuration" "escalation_channel" {
  count = local.has_critical_triggers ? 1 : 0
  
  name        = "escalation-for-${data.opensearch_detector.linux_detector_by_type.name}"
  config_type = "email"
  description = "Escalation channel for critical alerts from ${data.opensearch_detector.linux_detector_by_type.name}"
  
  email {
    email_account_id = var.security_team_email_account
    recipient_list {
      recipient = "security-team-leads@company.com"
    }
  }
}

# Output information about detector triggers
output "detector_triggers_summary" {
  value = {
    total_triggers = length(data.opensearch_detector.linux_detector_by_type.triggers)
    critical_triggers = length([
      for trigger in data.opensearch_detector.linux_detector_by_type.triggers : trigger
      if trigger.severity == "1"
    ])
    high_triggers = length([
      for trigger in data.opensearch_detector.linux_detector_by_type.triggers : trigger  
      if trigger.severity == "2"
    ])
    trigger_names = [
      for trigger in data.opensearch_detector.linux_detector_by_type.triggers : trigger.name
    ]
  }
}

# Example: Reference detector rules in other configurations
locals {
  # Extract all rule IDs from detector inputs
  all_rule_ids = flatten([
    for input in data.opensearch_detector.linux_detector_by_type.inputs : [
      for detector_input in input.detector_input : concat(
        [for rule in try(detector_input.custom_rules, []) : rule.id],
        [for rule in try(detector_input.pre_packaged_rules, []) : rule.id]
      )
    ]
  ])
}

output "detector_rule_ids" {
  description = "All rule IDs used by the Linux detector"
  value       = local.all_rule_ids
}