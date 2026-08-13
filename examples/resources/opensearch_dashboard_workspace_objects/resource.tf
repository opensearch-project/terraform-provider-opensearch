provider "opensearch" {
  url            = "http://127.0.0.1:9200"
  dashboards_url = "http://127.0.0.1:5601"
}

resource "opensearch_dashboard_workspace" "logs" {
  name        = "Logs"
  description = "Production log analysis"
  features    = ["use-case-observability"]
}

resource "opensearch_dashboard_workspace_objects" "logs_index_pattern" {
  workspace_id = opensearch_dashboard_workspace.logs.id

  saved_object {
    type = "index-pattern"
    id   = "logs-*"
  }
}
