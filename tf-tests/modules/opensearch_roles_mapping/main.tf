resource "opensearch_roles_mapping" "this" {
  role_name     = var.role_name
  users         = var.users
  backend_roles = var.backend_roles
}
