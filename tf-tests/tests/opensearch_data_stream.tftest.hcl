provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_index_template" {
  state_key = "create_index_template"

  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "test-data-stream-template"
    body = jsonencode({
      index_patterns = ["test-data-stream*"]
      data_stream    = {}
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == "test-data-stream-template"
    error_message = "Index template name should be test-data-stream-template"
  }
}

run "create_data_stream" {
  state_key = "create_data_stream"

  module {
    source = "./modules/opensearch_data_stream"
  }

  variables {
    name = "test-data-stream"
  }

  assert {
    condition     = opensearch_data_stream.this.name == "test-data-stream"
    error_message = "Data stream name should be test-data-stream"
  }
}
