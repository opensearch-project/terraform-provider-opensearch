provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "read_host_info" {
  module {
    source = "./modules/opensearch_host"
  }

  variables {
    active = true
  }

  assert {
    condition     = data.opensearch_host.this.url != ""
    error_message = "Host URL should not be empty"
  }

  assert {
    condition     = data.opensearch_host.this.id != ""
    error_message = "Host ID should not be empty"
  }
}
