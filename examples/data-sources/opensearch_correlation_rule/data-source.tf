# Retrieve an existing correlation rule by ID
data "opensearch_correlation_rule" "existing" {
  rule_id = "MLm_zZ0BghYda8SUa-hc"
}

# Use the correlation rule data in outputs
output "correlation_rule_name" {
  value = data.opensearch_correlation_rule.existing.name
}

output "correlation_queries" {
  value = data.opensearch_correlation_rule.existing.correlate
}

output "time_window_minutes" {
  value = data.opensearch_correlation_rule.existing.time_window / 60000
}

# Reference data from a created resource
resource "opensearch_correlation_rule" "new" {
  name        = "New Correlation Rule"
  time_window = 300000

  correlate {
    index    = "vpc_flow"
    query    = "suspicious_traffic:true"
    category = "network"
  }

  correlate {
    index    = "app_logs"
    query    = "error:true"
    category = "others_application"
  }
}

data "opensearch_correlation_rule" "created" {
  rule_id = opensearch_correlation_rule.new.id
}

# Use in another resource configuration
resource "opensearch_monitor" "correlation_monitor" {
  body = jsonencode({
    name = "Monitor for ${data.opensearch_correlation_rule.created.name}"
    type = "monitor"
    # ... other monitor configuration
  })
}