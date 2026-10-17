# Integration Test: Multi-Tenant Dashboard Environment
# Tests SaaS-style multi-tenancy with data isolation:
# 1. Dashboard tenants (Customer A & B)
# 2. Index template (data streams support with metrics mappings)
# 3. Data streams (per-customer metrics)
# 4. Dashboard objects (index pattern saved object)
# 5. Tenant-specific roles
# 6. Users with attributes
# 7. Roles mappings
# 8. Ingest pipeline (tenant routing)

provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

# ---------------------------------------------------------------------------
# 1. Tenant A
# ---------------------------------------------------------------------------
run "create_tenant_a" {

  state_key = "tenant-a"

  module {
    source = "./modules/opensearch_dashboard_tenant"
  }

  variables {
    tenant_name = "integration-customer-a"
    description = "Customer A tenant"
  }

  assert {
    condition     = opensearch_dashboard_tenant.this.tenant_name == "integration-customer-a"
    error_message = "Tenant A name should match"
  }
}

# ---------------------------------------------------------------------------
# 2. Tenant B
# ---------------------------------------------------------------------------
run "create_tenant_b" {

  state_key = "tenant-b"

  module {
    source = "./modules/opensearch_dashboard_tenant"
  }

  variables {
    tenant_name = "integration-customer-b"
    description = "Customer B tenant"
  }

  assert {
    condition     = opensearch_dashboard_tenant.this.tenant_name == "integration-customer-b"
    error_message = "Tenant B name should match"
  }
}

# ---------------------------------------------------------------------------
# 3. Composable index template for metrics data streams
# ---------------------------------------------------------------------------
run "create_metrics_template" {
  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "integration-metrics-template"
    body = jsonencode({
      index_patterns = ["integration-metrics-*"]
      priority       = 250
      data_stream    = {}
      template = {
        settings = {
          number_of_shards   = 1
          number_of_replicas = 0
        }
        mappings = {
          properties = {
            "@timestamp" = { type = "date" }
            cpu_percent  = { type = "float" }
            memory_used  = { type = "long" }
            tenant       = { type = "keyword" }
          }
        }
      }
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == "integration-metrics-template"
    error_message = "Index template name should match"
  }
}

# ---------------------------------------------------------------------------
# 4. Data stream for Customer A
# ---------------------------------------------------------------------------
run "create_data_stream_a" {

  state_key = "tenant-a"

  module {
    source = "./modules/opensearch_data_stream"
  }

  variables {
    name = "integration-metrics-customer-a"
  }

  assert {
    condition     = opensearch_data_stream.this.name == "integration-metrics-customer-a"
    error_message = "Data stream A name should match"
  }
}

# ---------------------------------------------------------------------------
# 5. Data stream for Customer B
# ---------------------------------------------------------------------------
run "create_data_stream_b" {

  state_key = "tenant-b"

  module {
    source = "./modules/opensearch_data_stream"
  }

  variables {
    name = "integration-metrics-customer-b"
  }

  assert {
    condition     = opensearch_data_stream.this.name == "integration-metrics-customer-b"
    error_message = "Data stream B name should match"
  }
}

# ---------------------------------------------------------------------------
# 6. Dashboard saved object — index pattern
# ---------------------------------------------------------------------------
run "create_index_pattern" {
  module {
    source = "./modules/opensearch_dashboard_object"
  }

  variables {
    body = jsonencode([{
      _id   = "index-pattern:integration-metrics"
      _type = "index-pattern"
      _source = {
        title         = "integration-metrics-*"
        timeFieldName = "@timestamp"
        fields        = jsonencode([]) # required by the provider
      }
    }])
  }

  assert {
    condition     = opensearch_dashboard_object.this.id != ""
    error_message = "Dashboard object should be created"
  }
}

# ---------------------------------------------------------------------------
# 7. Tenant-specific role A
# ---------------------------------------------------------------------------
run "create_role_a" {

  state_key = "tenant-a"

  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = "integration-customer-a-role"
    description         = "Read/write access to Customer A metrics"
    cluster_permissions = []
    index_permissions = [
      {
        index_patterns          = ["integration-metrics-customer-a"]
        allowed_actions         = ["read", "write", "create_index"]
        document_level_security = ""
        field_level_security    = []
        masked_fields           = []
      }
    ]
    tenant_permissions = [
      {
        tenant_patterns = ["integration-customer-a"]
        allowed_actions = ["kibana_all_write"]
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "integration-customer-a-role"
    error_message = "Role A name should match"
  }
}

# ---------------------------------------------------------------------------
# 8. Tenant-specific role B
# ---------------------------------------------------------------------------
run "create_role_b" {

  state_key = "tenant-b"

  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = "integration-customer-b-role"
    description         = "Read/write access to Customer B metrics"
    cluster_permissions = []
    index_permissions = [
      {
        index_patterns          = ["integration-metrics-customer-b"]
        allowed_actions         = ["read", "write", "create_index"]
        document_level_security = ""
        field_level_security    = []
        masked_fields           = []
      }
    ]
    tenant_permissions = [
      {
        tenant_patterns = ["integration-customer-b"]
        allowed_actions = ["kibana_all_write"]
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "integration-customer-b-role"
    error_message = "Role B name should match"
  }
}

# ---------------------------------------------------------------------------
# 9. User for tenant A
# ---------------------------------------------------------------------------
run "create_user_a" {

  state_key = "tenant-a"

  module {
    source = "./modules/opensearch_user"
  }

  variables {
    username    = "integration-customer-a-user"
    password    = "TenantA\u0021Passw0rd"
    description = "User with access to Customer A"
    attributes = {
      tenant = "customer-a"
    }
  }

  assert {
    condition     = opensearch_user.this.username == "integration-customer-a-user"
    error_message = "User A username should match"
  }
}

# ---------------------------------------------------------------------------
# 10. User for tenant B
# ---------------------------------------------------------------------------
run "create_user_b" {
  state_key = "tenant-b"

  module {
    source = "./modules/opensearch_user"
  }

  variables {
    username    = "integration-customer-b-user"
    password    = "TenantB\u0021Passw0rd"
    description = "User with access to Customer B"
    attributes = {
      tenant = "customer-b"
    }
  }

  assert {
    condition     = opensearch_user.this.username == "integration-customer-b-user"
    error_message = "User B username should match"
  }
}

# ---------------------------------------------------------------------------
# 11. Ingest pipeline for tenant routing
# ---------------------------------------------------------------------------
run "create_ingest_pipeline" {
  module {
    source = "./modules/opensearch_ingest_pipeline"
  }

  variables {
    name = "integration-tenant-routing"
    body = jsonencode({
      description = "Route metrics documents and enrich with tenant metadata"
      processors = [
        {
          set = {
            field = "ingest_timestamp"
            value = "{{_ingest.timestamp}}"
          }
        },
        {
          script = {
            lang   = "painless"
            source = "ctx.tenant = ctx._index.contains('customer-a') ? 'customer-a' : 'customer-b';"
          }
        },
        {
          convert = {
            field = "cpu_percent"
            type  = "float"
          }
        }
      ]
    })
  }

  assert {
    condition     = opensearch_ingest_pipeline.this.name == "integration-tenant-routing"
    error_message = "Ingest pipeline name should match"
  }
}
