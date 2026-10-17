output "id" {
  description = "ID of the index template"
  value       = opensearch_composable_index_template.this.id
}

output "name" {
  description = "Name of the index template"
  value       = opensearch_composable_index_template.this.name
}
