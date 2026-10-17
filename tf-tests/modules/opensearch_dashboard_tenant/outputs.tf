output "id" {
  description = "ID of the resource"
  value       = opensearch_dashboard_tenant.this.id
}

output "tenant_name" {
  description = "Tenant name"
  value       = opensearch_dashboard_tenant.this.tenant_name
}
