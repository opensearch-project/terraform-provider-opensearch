# Minimal ML Model Group (defaults to private access mode)
resource "opensearch_ml_model_group" "minimal" {
  name = "minimal"
}

# ML Model Group with description
resource "opensearch_ml_model_group" "with_description" {
  name        = "with_description"
  description = "ML Model Group with 'description' attribute"
}

# ML Model Group with private access mode (explicit)
resource "opensearch_ml_model_group" "private" {
  name        = "private"
  access_mode = "private"
}

# ML Model Group with restricted access and specific backend roles
resource "opensearch_ml_model_group" "restricted_with_backend_roles" {
  name          = "restricted_with_backend_roles"
  access_mode   = "restricted"
  backend_roles = ["ml_full_access"]
}

# ML Model Group with restricted access and all backend roles
# Note: This may not work for admin users due to OpenSearch API restrictions
resource "opensearch_ml_model_group" "restricted_with_add_all_backend_roles" {
  name                  = "restricted_with_add_all_backend_roles"
  access_mode           = "restricted"
  add_all_backend_roles = true
}
