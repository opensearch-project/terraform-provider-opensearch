resource "opensearch_audit_config" "this" {
  enabled = var.enabled

  audit {
    enable_rest                   = var.enable_rest
    enable_transport              = var.enable_transport
    ignore_users                  = var.ignore_users
    ignore_requests               = var.ignore_requests
    disabled_rest_categories      = var.disabled_rest_categories
    disabled_transport_categories = var.disabled_transport_categories
  }

  dynamic "compliance" {
    for_each = var.compliance_enabled ? [1] : []
    content {
      enabled               = var.compliance_enabled
      write_watched_indices = var.compliance_write_watched_indices
    }
  }
}
