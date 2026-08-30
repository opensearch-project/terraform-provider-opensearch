# Integration Test: Log Ingestion Pipeline
# Tests a complete log ingestion workflow with ISM:
# 1. Component template (common log mappings)
# 2. Index template (composable with component)
# 3. ISM policy (Hot→Warm→Delete lifecycle with ism_template)
# 4. Ingest pipeline (log enrichment)
# 5. Bootstrap index with alias (integration-log-application-000001)
# 6. Security role (log_reader)

provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

# ---------------------------------------------------------------------------
# 1. Component template — common log mappings
# ---------------------------------------------------------------------------
run "create_component_template" {
  module {
    source = "./modules/opensearch_component_template"
  }

  variables {
    name = "integration-log-mappings"
    body = jsonencode({
      template = {
        mappings = {
          properties = {
            "@timestamp" = { type = "date" }
            level        = { type = "keyword" }
            message      = { type = "text" }
            service      = { type = "keyword" }
            request_id   = { type = "keyword" }
          }
        }
      }
    })
  }

  assert {
    condition     = opensearch_component_template.this.name == "integration-log-mappings"
    error_message = "Component template name should match"
  }
}

# ---------------------------------------------------------------------------
# 2. Composable index template referencing the component template
# ---------------------------------------------------------------------------
run "create_index_template" {
  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "integration-log-template"
    body = jsonencode({
      index_patterns = ["integration-log-application-*"]
      priority       = 200
      composed_of    = [run.create_component_template.name]
      template = {
        settings = {
          number_of_shards   = 1
          number_of_replicas = 0
        }
        mappings = {
          _source = { enabled = true }
        }
      }
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == "integration-log-template"
    error_message = "Index template name should match"
  }
}

# ---------------------------------------------------------------------------
# 3. ISM policy with Hot→Warm→Delete and ism_template
# ---------------------------------------------------------------------------
run "create_ism_policy" {
  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = "integration-log-lifecycle"
    body = jsonencode({
      policy = {
        description   = "Hot-Warm-Delete policy for application logs"
        default_state = "hot"
        states = [
          {
            name = "hot"
            actions = [
              {
                rollover = {
                  min_index_age          = "1d"
                  min_primary_shard_size = "50gb"
                }
              }
            ]
            transitions = [
              {
                state_name = "warm"
                conditions = { min_index_age = "3d" }
              }
            ]
          },
          {
            name = "warm"
            actions = [
              {
                force_merge = { max_num_segments = 1 }
              }
            ]
            transitions = [
              {
                state_name = "delete"
                conditions = { min_index_age = "30d" }
              }
            ]
          },
          {
            name = "delete"
            actions = [
              {
                delete = {}
              }
            ]
            transitions = []
          }
        ]
        ism_template = {
          index_patterns = ["integration-log-application-*"]
          priority       = 200
        }
      }
    })
  }

  assert {
    condition     = opensearch_ism_policy.this.policy_id == "integration-log-lifecycle"
    error_message = "ISM policy ID should match"
  }
}

# ---------------------------------------------------------------------------
# 4. Ingest pipeline — log enrichment
# ---------------------------------------------------------------------------
run "create_ingest_pipeline" {
  module {
    source = "./modules/opensearch_ingest_pipeline"
  }

  variables {
    name = "integration-log-enrichment"
    body = jsonencode({
      description = "Enrich application log entries"
      processors = [
        {
          set = {
            field = "ingest_timestamp"
            value = "{{_ingest.timestamp}}"
          }
        },
        {
          lowercase = {
            field = "level"
          }
        },
        {
          date = {
            field    = "@timestamp"
            formats  = ["ISO8601"]
            timezone = "UTC"
          }
        }
      ]
    })
  }

  assert {
    condition     = opensearch_ingest_pipeline.this.name == "integration-log-enrichment"
    error_message = "Ingest pipeline name should match"
  }
}

# ---------------------------------------------------------------------------
# 5. Bootstrap index with alias for rollover
# ---------------------------------------------------------------------------
run "create_bootstrap_index" {
  module {
    source = "./modules/opensearch_index"
  }

  variables {
    name               = "integration-log-application-000001"
    number_of_shards   = 1
    number_of_replicas = 0
    aliases = {
      "integration-log-application" = {
        is_write_index = true
      }
    }
  }

  assert {
    condition     = opensearch_index.this.name == "integration-log-application-000001"
    error_message = "Bootstrap index name should match"
  }
}

# ---------------------------------------------------------------------------
# 6. Security role — log_reader
# ---------------------------------------------------------------------------
run "create_role" {
  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name           = "integration-log-reader"
    description         = "Read-only role for log indices"
    cluster_permissions = ["cluster:monitor/health"]
    index_permissions = [
      {
        index_patterns          = ["integration-log-application-*"]
        allowed_actions         = ["read", "indices:monitor/stats"]
        document_level_security = ""
        field_level_security    = []
        masked_fields           = []
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "integration-log-reader"
    error_message = "Role name should match"
  }
}
