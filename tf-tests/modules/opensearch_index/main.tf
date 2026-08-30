resource "opensearch_index" "this" {
  name               = var.name
  number_of_shards   = var.number_of_shards
  number_of_replicas = var.number_of_replicas
  mappings           = var.mappings
}
