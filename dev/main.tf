# Security: User
resource "opensearch_user" "user" {
  username    = "dev-sandbox-user"
  password    = "DevSandbox123!"
  description = "Developer sandbox application user"
}

# Security: Role
resource "opensearch_role" "role" {
  role_name   = "dev-sandbox-reader"
  description = "Read-only access to dev-sandbox indices"

  index_permissions {
    index_patterns  = ["dev-sandbox-*"]
    allowed_actions = ["read", "get", "search", "indices:monitor/stats"]
  }
}

# Security: Role Mapping
resource "opensearch_roles_mapping" "role_mapping" {
  role_name = opensearch_role.role.id
  users     = [opensearch_user.user.id]
}

# Index
resource "opensearch_index" "index" {
  name               = "dev-sandbox-index"
  number_of_shards   = "1"
  number_of_replicas = "0"
  mappings = jsonencode({
    properties = {
      timestamp = { type = "date" }
      message   = { type = "text" }
    }
  })
}
