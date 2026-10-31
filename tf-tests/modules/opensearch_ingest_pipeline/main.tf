resource "opensearch_ingest_pipeline" "this" {
  name = var.name
  body = var.body
}
