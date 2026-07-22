resource "opensearch_dashboard" "example" {
  # An ndjson export from OpenSearch Dashboards' Saved Objects Management UI
  # (Stack Management > Saved Objects > Export), or hand-written ndjson.
  source = file("${path.module}/export.ndjson")
}
