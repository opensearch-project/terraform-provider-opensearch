output "id" {
  description = "ID of the resource"
  value       = opensearch_user.this.id
}

output "name" {
  description = "Name of the resource"
  value       = opensearch_user.this.username
}

output "username" {
  description = "Username"
  value       = opensearch_user.this.username
}

output "backend_roles" {
  description = "Backend roles"
  value       = opensearch_user.this.backend_roles
}

output "attributes" {
  description = "Attributes"
  value       = opensearch_user.this.attributes
}

output "description" {
  description = "User Description"
  value       = opensearch_user.this.description
}
