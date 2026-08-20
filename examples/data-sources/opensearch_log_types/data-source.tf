# Get all log types
data "opensearch_log_types" "all" {}

# Filter by source to get only custom log types
data "opensearch_log_types" "custom_only" {
  source = "Custom"
}

# Filter by category
data "opensearch_log_types" "security_logs" {
  category = "Security"
}

# Filter by name to get a specific log type
data "opensearch_log_types" "specific" {
  name = "custom-application-logs"
}

# Filter by ID to get a specific log type
data "opensearch_log_types" "by_id" {
  id = "lJ66SJ0Bz0C14FDnbleI"
}

# Example: Create a log type and then query it
resource "opensearch_log_type" "example" {
  name        = "my-custom-logs"
  description = "My custom log type"
  source      = "Custom"
  category    = "Applications"
}

data "opensearch_log_types" "verify" {
  name       = opensearch_log_type.example.name
  depends_on = [opensearch_log_type.example]
}

output "log_type_id" {
  value = data.opensearch_log_types.verify.log_types[0].id
}
