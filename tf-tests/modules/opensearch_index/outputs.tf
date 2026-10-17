output "name" {
  description = "Name of the index"
  value       = opensearch_index.this.name
}

output "id" {
  description = "ID of the index"
  value       = opensearch_index.this.id
}
