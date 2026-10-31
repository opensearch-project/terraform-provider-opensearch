resource "opensearch_role" "this" {
  role_name           = var.role_name
  description         = var.description
  cluster_permissions = var.cluster_permissions

  dynamic "index_permissions" {
    for_each = var.index_permissions
    content {
      index_patterns          = index_permissions.value.index_patterns
      allowed_actions         = index_permissions.value.allowed_actions
      document_level_security = index_permissions.value.document_level_security != "" ? index_permissions.value.document_level_security : null
      field_level_security    = length(index_permissions.value.field_level_security) > 0 ? index_permissions.value.field_level_security : null
      masked_fields           = length(index_permissions.value.masked_fields) > 0 ? index_permissions.value.masked_fields : null
    }
  }

  dynamic "tenant_permissions" {
    for_each = var.tenant_permissions
    content {
      tenant_patterns = tenant_permissions.value.tenant_patterns
      allowed_actions = tenant_permissions.value.allowed_actions
    }
  }
}
