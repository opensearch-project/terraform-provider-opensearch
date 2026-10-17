resource "opensearch_ml_model_group" "this" {
  name                  = var.name
  description           = var.description
  access_mode           = var.access_mode
  backend_roles         = var.backend_roles
  add_all_backend_roles = var.add_all_backend_roles
}