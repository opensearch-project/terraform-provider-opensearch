# Create a role
resource "opensearch_role" "writer" {
  role_name   = "logs_writer"
  description = "Logs writer role"

  cluster_permissions = ["*"]

  index_permissions {
    index_patterns  = ["logstash-*"]
    allowed_actions = ["write"]
  }

  tenant_permissions {
    tenant_patterns = ["logstash-*"]
    allowed_actions = ["write"]
  }
}

# To set document level permissions:
resource "opensearch_role" "writer" {
  role_name = "foo_writer"

  cluster_permissions = ["*"]

  index_permissions {
    index_patterns          = ["pub*"]
    allowed_actions         = ["read"]
    document_level_security = "{\"term\": { \"readable_by\": \"$${user.name}\"}}"
  }
}
