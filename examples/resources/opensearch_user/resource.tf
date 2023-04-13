# Create a user
resource "opensearch_user" "mapper" {
  username    = "app-reasdder"
  password    = "SuperSekret123!"
  description = "a reader role for our app"
}

# And a full user, role and role mapping example:
resource "opensearch_role" "reader" {
  role_name   = "app_reader"
  description = "App Reader Role"

  index_permissions {
    index_patterns  = ["app-*"]
    allowed_actions = ["get", "read", "search"]
  }
}

resource "opensearch_user" "reader" {
  username = "app-reader"
  password = var.password
}

resource "opensearch_roles_mapping" "reader" {
  role_name   = opensearch_role.reader.id
  description = "App Reader Role"
  users       = [opensearch_user.reader.id]
}
