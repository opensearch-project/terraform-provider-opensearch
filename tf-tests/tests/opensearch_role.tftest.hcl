provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_role" {

  state_key = "test-role"

  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = "test-role"
    description         = "Test role"
    cluster_permissions = ["cluster:monitor/health"]
    index_permissions = [
      {
        index_patterns  = ["logs-*"]
        allowed_actions = ["read"]
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "test-role"
    error_message = "Role name should be test-role"
  }
}

run "update_role" {

  state_key = "test-role"

  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = run.create_role.role_name
    description         = "Updated test role"
    cluster_permissions = ["cluster:monitor/health", "cluster:monitor/state"]
    index_permissions = [
      {
        index_patterns  = ["logs-*", "metrics-*"]
        allowed_actions = ["read", "write"]
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "test-role"
    error_message = "Role name should remain test-role"
  }
}

run "create_full_role" {

  state_key = "test-full-role"

  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = "test-full-role"
    description         = "Full role with DLS, FLS and tenant permissions"
    cluster_permissions = ["cluster:monitor/health", "cluster:monitor/state"]
    index_permissions = [
      {
        index_patterns       = ["logs-*"]
        allowed_actions      = ["read", "indices:admin/mapping/put"]
        field_level_security = ["@timestamp", "message", "level"]
        document_level_security = jsonencode({
          term = { level = "error" }
        })
      }
    ]
    tenant_permissions = [
      {
        tenant_patterns = ["test-tenant"]
        allowed_actions = ["kibana_all_read"]
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "test-full-role"
    error_message = "Role name should be test-full-role"
  }
}

