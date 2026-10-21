resource "opensearch_log_type" "example" {
  name        = "custom-application-logs"
  description = "Custom log type for application logs"
  source      = "Custom"
  category    = "Applications"
}
