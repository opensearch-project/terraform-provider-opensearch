resource "opensearch_composable_index_template" "this" {
  name = var.name
  body = var.body
}
