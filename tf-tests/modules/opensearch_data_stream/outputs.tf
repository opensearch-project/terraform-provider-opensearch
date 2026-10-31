output "id" {
  description = "ID of the data stream"
  value       = opensearch_data_stream.this.id
}

output "name" {
  description = "Name of the data stream"
  value       = opensearch_data_stream.this.name
}
