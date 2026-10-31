provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "enable_audit" {
  module {
    source = "./modules/opensearch_audit_config"
  }

  variables {
    enabled = true
  }

  assert {
    condition     = opensearch_audit_config.this.enabled == true
    error_message = "Audit config should be enabled"
  }
}

run "full_audit_config" {
  module {
    source = "./modules/opensearch_audit_config"
  }

  variables {
    enabled                          = true
    compliance_enabled               = true
    compliance_write_watched_indices = ["logs-*", "metrics-*"]
    enable_rest                      = true
    enable_transport                 = true
    ignore_users                     = ["kibanaserver"]
    ignore_requests                  = []
    disabled_rest_categories         = ["AUTHENTICATED", "GRANTED_PRIVILEGES"]
    disabled_transport_categories    = ["AUTHENTICATED", "GRANTED_PRIVILEGES"]
  }

  assert {
    condition     = opensearch_audit_config.this.enabled == true
    error_message = "Audit config should be enabled"
  }
}
