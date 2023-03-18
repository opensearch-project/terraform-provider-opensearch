# Create a tenant
resource "opensearch_dashboard_tenant" "test" {
  tenant_name = "test"
  description = "test tenant"
}
