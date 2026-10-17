provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_component_template" {
  state_key = "create_component_template"

  module {
    source = "./modules/opensearch_component_template"
  }

  variables {
    name = "test-component"
    body = jsonencode({
      template = {
        settings = {
          number_of_shards   = 1
          number_of_replicas = 0
        }
        mappings = {
          properties = {
            host_name = {
              type = "keyword"
            }
            created_at = {
              type = "date"
            }
          }
        }
      }
    })
  }

  assert {
    condition     = opensearch_component_template.this.name == "test-component"
    error_message = "Component template name should be test-component"
  }
}

run "create_component_template_full" {
  state_key = "create_component_template_full"

  module {
    source = "./modules/opensearch_component_template"
  }

  variables {
    name = "test-component-full"
    body = jsonencode({
      template = {
        settings = {
          number_of_shards   = 2
          number_of_replicas = 1
          refresh_interval   = "10s"
          index = {
            codec = "best_compression"
          }
        }
        mappings = {
          dynamic = "strict"
          properties = {
            host_name = {
              type         = "keyword"
              ignore_above = 256
            }
            created_at = {
              type   = "date"
              format = "strict_date_optional_time||epoch_millis"
            }
            tags = {
              type = "keyword"
            }
          }
        }
        aliases = {
          test_alias = {}
        }
      }
      version = 1
      _meta = {
        description = "Full component template"
      }
    })
  }

  assert {
    condition     = opensearch_component_template.this.name == "test-component-full"
    error_message = "Component template name should be test-component-full"
  }
}

run "read_component_template_body" {
  command   = plan
  state_key = "create_component_template_full"

  module {
    source = "./modules/opensearch_component_template"
  }

  variables {
    name = "test-component-full"
    body = jsonencode({
      template = {
        settings = {
          number_of_shards   = 2
          number_of_replicas = 1
          refresh_interval   = "10s"
          index = {
            codec = "best_compression"
          }
        }
        mappings = {
          dynamic = "strict"
          properties = {
            host_name = {
              type         = "keyword"
              ignore_above = 256
            }
            created_at = {
              type   = "date"
              format = "strict_date_optional_time||epoch_millis"
            }
            tags = {
              type = "keyword"
            }
          }
        }
        aliases = {
          test_alias = {}
        }
      }
      version = 1
      _meta = {
        description = "Full component template"
      }
    })
  }

  assert {
    condition     = jsondecode(opensearch_component_template.this.body)["template"]["settings"]["number_of_shards"] == 2
    error_message = "Number of shards should be 2"
  }

  assert {
    condition     = jsondecode(opensearch_component_template.this.body)["template"]["mappings"]["dynamic"] == "strict"
    error_message = "Dynamic mapping should be strict"
  }

  assert {
    condition     = jsondecode(opensearch_component_template.this.body)["template"]["mappings"]["properties"]["host_name"]["type"] == "keyword"
    error_message = "host_name type should be keyword"
  }

  assert {
    condition     = jsondecode(opensearch_component_template.this.body)["version"] == 1
    error_message = "Version should be 1"
  }
}

run "update_component_template" {
  state_key = "create_component_template"

  module {
    source = "./modules/opensearch_component_template"
  }

  variables {
    name = run.create_component_template.name
    body = jsonencode({
      template = {
        settings = {
          number_of_shards   = 2
          number_of_replicas = 1
        }
        mappings = {
          properties = {
            host_name = {
              type = "keyword"
            }
            created_at = {
              type = "date"
            }
          }
        }
      }
    })
  }

  assert {
    condition     = opensearch_component_template.this.name == run.create_component_template.name
    error_message = "Component template name should match the previous run"
  }
}
