resource "opensearch_user" "this" {
  username      = var.username
  password      = var.password
  description   = var.description
  attributes    = var.attributes
  backend_roles = var.backend_roles
}
