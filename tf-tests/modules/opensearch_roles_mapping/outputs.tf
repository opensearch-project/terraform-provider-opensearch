output "id" {
  description = "ID of the resource"
  value       = opensearch_roles_mapping.this.id
}

output "role_name" {
  description = "Role name"
  value       = opensearch_roles_mapping.this.role_name
}
