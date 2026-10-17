provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "setup_role" {
  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = "test-role"
    description         = "Test role for mapping"
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
    error_message = "Setup role name should be test-role"
  }
}

run "create_mapping" {
  module {
    source = "./modules/opensearch_roles_mapping"
  }

  variables {
    role_name     = run.setup_role.role_name
    users         = ["test-user"]
    backend_roles = ["admin-group"]
  }

  assert {
    condition     = opensearch_roles_mapping.this.role_name == "test-role"
    error_message = "Role name should be test-role"
  }
}

run "update_mapping" {
  module {
    source = "./modules/opensearch_roles_mapping"
  }

  variables {
    role_name     = run.setup_role.role_name
    users         = ["test-user", "additional-user"]
    backend_roles = ["admin-group", "power-user-group"]
  }

  assert {
    condition     = opensearch_roles_mapping.this.role_name == "test-role"
    error_message = "Role name should remain test-role"
  }
}

