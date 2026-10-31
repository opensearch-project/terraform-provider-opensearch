output "id" {
  description = "ID of the resource"
  value       = opensearch_role.this.id
}

output "name" {
  description = "Name of the resource"
  value       = opensearch_role.this.role_name
}

output "role_name" {
  description = "Role name"
  value       = opensearch_role.this.role_name
}
