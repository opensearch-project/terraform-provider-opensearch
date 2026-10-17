provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_template" {
  state_key = "create_template"

  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "test-template"
    body = jsonencode({
      index_patterns = ["logs-*"]
      template = {
        settings = {
          number_of_shards = 1
        }
      }
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == "test-template"
    error_message = "Index template name should be test-template"
  }
}

run "create_template_full" {
  state_key = "create_template_full"

  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "test-template-full"
    body = jsonencode({
      index_patterns = ["logs-*", "metrics-*", "traces-*"]
      priority       = 500
      version        = 1
      composed_of    = []
      template = {
        settings = {
          number_of_shards   = 2
          number_of_replicas = 1
          refresh_interval   = "5s"
        }
        mappings = {
          dynamic_templates = [
            {
              strings_as_keywords = {
                match_mapping_type = "string"
                mapping = {
                  type = "keyword"
                }
              }
            }
          ]
          properties = {
            timestamp = {
              type   = "date"
              format = "strict_date_optional_time||epoch_millis"
            }
            message = {
              type = "text"
            }
            level = {
              type = "keyword"
            }
          }
        }
      }
      _meta = {
        description = "Full test template"
      }
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == "test-template-full"
    error_message = "Index template name should be test-template-full"
  }
}

run "read_template_body" {
  command   = plan
  state_key = "create_template_full"

  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "test-template-full"
    body = jsonencode({
      index_patterns = ["logs-*", "metrics-*", "traces-*"]
      priority       = 500
      version        = 1
      composed_of    = []
      template = {
        settings = {
          number_of_shards   = 2
          number_of_replicas = 1
          refresh_interval   = "5s"
        }
        mappings = {
          dynamic_templates = [
            {
              strings_as_keywords = {
                match_mapping_type = "string"
                mapping = {
                  type = "keyword"
                }
              }
            }
          ]
          properties = {
            timestamp = {
              type   = "date"
              format = "strict_date_optional_time||epoch_millis"
            }
            message = {
              type = "text"
            }
            level = {
              type = "keyword"
            }
          }
        }
      }
      _meta = {
        description = "Full test template"
      }
    })
  }

  assert {
    condition     = jsondecode(opensearch_composable_index_template.this.body)["index_patterns"][0] == "logs-*"
    error_message = "First index pattern should be logs-*"
  }

  assert {
    condition     = jsondecode(opensearch_composable_index_template.this.body)["priority"] == 500
    error_message = "Priority should be 500"
  }

  assert {
    condition     = jsondecode(opensearch_composable_index_template.this.body)["template"]["settings"]["number_of_shards"] == 2
    error_message = "Number of shards should be 2"
  }

  assert {
    condition     = jsondecode(opensearch_composable_index_template.this.body)["template"]["mappings"]["properties"]["level"]["type"] == "keyword"
    error_message = "Level mapping type should be keyword"
  }
}

run "update_template" {
  state_key = "create_template"

  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = run.create_template.name
    body = jsonencode({
      index_patterns = ["logs-*", "metrics-*"]
      priority       = 100
      template = {
        settings = {
          number_of_shards = 2
        }
      }
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == run.create_template.name
    error_message = "Index template name should match the previous run"
  }
}
