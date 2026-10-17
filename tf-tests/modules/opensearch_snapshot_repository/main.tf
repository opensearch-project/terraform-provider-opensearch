resource "opensearch_snapshot_repository" "this" {
  name     = var.name
  type     = var.type
  settings = var.settings
}
