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

# ML Commons: MCP tool
#
# Off by default, because it needs an OpenSearch 3.1+ cluster with the MCP server enabled,
# while the sandbox defaults to 2.x. To exercise it:
#
#   make down
#   OSS_IMAGE=opensearchproject/opensearch:3 \
#   OSS_ENV_VAR="plugins.ml_commons.mcp_server_enabled=true" make dev-up
#   TF_VAR_enable_mcp_tool=true make dev-apply
resource "opensearch_ml_mcp_tool" "mcp_tool" {
  count = var.enable_mcp_tool ? 1 : 0

  name        = "ListIndexTool"
  type        = "ListIndexTool"
  description = "Lists all indices in the cluster"

  attributes = jsonencode({
    input_schema = {
      type = "object"
      properties = {
        indices = {
          type        = "array"
          items       = { type = "string" }
          description = "OpenSearch index name list, separated by comma"
        }
      }
    }
  })
}
