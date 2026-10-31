provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_tenant" {
  module {
    source = "./modules/opensearch_dashboard_tenant"
  }

  variables {
    tenant_name = "test-tenant"
    description = "Test tenant"
  }

  assert {
    condition     = opensearch_dashboard_tenant.this.tenant_name == "test-tenant"
    error_message = "Tenant name should be test-tenant"
  }
}

run "update_tenant" {
  module {
    source = "./modules/opensearch_dashboard_tenant"
  }

  variables {
    tenant_name = run.create_tenant.tenant_name
    description = "Updated tenant description"
  }

  assert {
    condition     = opensearch_dashboard_tenant.this.tenant_name == "test-tenant"
    error_message = "Tenant name should remain test-tenant"
  }
}

