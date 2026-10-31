output "id" {
  description = "ID of the resource"
  value       = opensearch_script.this.id
}

output "name" {
  description = "Name of the resource"
  value       = opensearch_script.this.script_id
}

output "script_id" {
  description = "Script ID"
  value       = opensearch_script.this.script_id
}
