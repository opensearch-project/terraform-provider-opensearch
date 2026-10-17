# Integration Test: Production Cluster Setup
# Tests production-ready configuration:
# 1. Cluster settings (watermarks, shard limits)
# 2. Snapshot repository (fs type)
# 3. SM policy (automated backups)
# 4. Audit config (compliance logging)
# 5. Component template (compliance metadata)
# 6. Index template (compliant indices)
# 7. Ingest pipeline (compliance enrichment)
# 8. ISM policy (7-year retention)
# 9. Admin roles (production-admin, backup-operator)

provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

# ---------------------------------------------------------------------------
# 1. Cluster settings
# ---------------------------------------------------------------------------
run "create_cluster_settings" {
  module {
    source = "./modules/opensearch_cluster_settings"
  }

  variables {
    cluster_max_shards_per_node                    = 1000
    cluster_routing_allocation_disk_watermark_low  = "85%"
    cluster_routing_allocation_disk_watermark_high = "90%"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_max_shards_per_node == 1000
    error_message = "Max shards per node should be 1000"
  }
}

# ---------------------------------------------------------------------------
# 2. Snapshot repository
# ---------------------------------------------------------------------------
run "create_snapshot_repository" {
  module {
    source = "./modules/opensearch_snapshot_repository"
  }

  variables {
    name = "integration-backup-repo"
    type = "fs"
    settings = {
      location = "/tmp/integration-snapshots"
      compress = "true"
    }
  }

  assert {
    condition     = opensearch_snapshot_repository.this.name == "integration-backup-repo"
    error_message = "Repository name should match"
  }
}

# ---------------------------------------------------------------------------
# 3. ISM policy — 7-year retention (approx 2555 days)
# ---------------------------------------------------------------------------
run "create_ism_policy" {
  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = "integration-7year-retention"
    body = jsonencode({
      policy = {
        description   = "7-year retention policy for compliance indices"
        default_state = "hot"
        states = [
          {
            name = "hot"
            actions = [
              {
                rollover = {
                  min_index_age          = "30d"
                  min_primary_shard_size = "50gb"
                }
              }
            ]
            transitions = [
              {
                state_name = "delete"
                conditions = { min_index_age = "2555d" }
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
          index_patterns = ["integration-compliance-*"]
          priority       = 500
        }
      }
    })
  }

  assert {
    condition     = opensearch_ism_policy.this.policy_id == "integration-7year-retention"
    error_message = "ISM policy ID should match"
  }
}

# ---------------------------------------------------------------------------
# 4. SM policy — automated backups
# ---------------------------------------------------------------------------
run "create_sm_policy" {
  module {
    source = "./modules/opensearch_sm_policy"
  }

  variables {
    policy_name = "integration-daily-backup"
    body = jsonencode({
      description = "Daily automated snapshot policy"
      creation = {
        schedule = {
          cron = {
            expression = "0 1 * * *"
            timezone   = "UTC"
          }
        }
      }
      deletion = {
        schedule = {
          cron = {
            expression = "0 2 * * *"
            timezone   = "UTC"
          }
        }
        condition = {
          max_count = 30
          min_count = 5
          min_age   = "7d"
        }
      }
      snapshot_config = {
        indices              = ["*"]
        repository           = "integration-backup-repo"
        include_global_state = false
        ignore_unavailable   = true
      }
    })
  }

  assert {
    condition     = opensearch_sm_policy.this.policy_name == "integration-daily-backup"
    error_message = "SM policy name should match"
  }
}

# ---------------------------------------------------------------------------
# 5. Audit config — compliance logging
# ---------------------------------------------------------------------------
run "create_audit_config" {
  module {
    source = "./modules/opensearch_audit_config"
  }

  variables {
    enabled                          = true
    enable_rest                      = true
    enable_transport                 = true
    ignore_users                     = ["kibanaserver"]
    ignore_requests                  = []
    disabled_rest_categories         = ["AUTHENTICATED", "GRANTED_PRIVILEGES"]
    disabled_transport_categories    = ["AUTHENTICATED", "GRANTED_PRIVILEGES"]
    compliance_enabled               = true
    compliance_write_watched_indices = ["integration-compliance-*"]
  }

  assert {
    condition     = opensearch_audit_config.this.enabled == true
    error_message = "Audit config should be enabled"
  }
}

# ---------------------------------------------------------------------------
# 6. Component template — compliance metadata
# ---------------------------------------------------------------------------
run "create_component_template" {
  module {
    source = "./modules/opensearch_component_template"
  }

  variables {
    name = "integration-compliance-metadata"
    body = jsonencode({
      template = {
        settings = {
          number_of_shards   = 1
          number_of_replicas = 0
        }
        mappings = {
          properties = {
            "@timestamp"       = { type = "date" }
            data_class         = { type = "keyword" }
            retention_class    = { type = "keyword" }
            owner              = { type = "keyword" }
            compliance_version = { type = "keyword" }
          }
        }
      }
    })
  }

  assert {
    condition     = opensearch_component_template.this.name == "integration-compliance-metadata"
    error_message = "Component template name should match"
  }
}

# ---------------------------------------------------------------------------
# 7. Index template — compliant indices composed with metadata component
# ---------------------------------------------------------------------------
run "create_index_template" {
  module {
    source = "./modules/opensearch_index_template"
  }

  variables {
    name = "integration-compliance-template"
    body = jsonencode({
      index_patterns = ["integration-compliance-*"]
      priority       = 500
      composed_of    = ["integration-compliance-metadata"]
      template = {
        settings = {
          number_of_shards   = 2
          number_of_replicas = 1
        }
      }
    })
  }

  assert {
    condition     = opensearch_composable_index_template.this.name == "integration-compliance-template"
    error_message = "Index template name should match"
  }
}

# ---------------------------------------------------------------------------
# 8. Ingest pipeline — compliance enrichment
# ---------------------------------------------------------------------------
run "create_ingest_pipeline" {
  module {
    source = "./modules/opensearch_ingest_pipeline"
  }

  variables {
    name = "integration-compliance-enrichment"
    body = jsonencode({
      description = "Enrich compliance documents with metadata"
      processors = [
        {
          set = {
            field = "compliance_version"
            value = "v1.0.0"
          }
        },
        {
          set = {
            field = "ingest_timestamp"
            value = "{{_ingest.timestamp}}"
          }
        },
        {
          script = {
            lang   = "painless"
            source = "ctx.data_class = ctx._index.contains('financial') ? 'financial' : 'general'"
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
    condition     = opensearch_ingest_pipeline.this.name == "integration-compliance-enrichment"
    error_message = "Pipeline name should match"
  }
}

# ---------------------------------------------------------------------------
# 9. Admin role — production admin
# ---------------------------------------------------------------------------
run "create_production_admin_role" {
  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name   = "integration-production-admin"
    description = "Production administrator with cluster and index access"
    cluster_permissions = [
      "cluster:*"
    ]
    index_permissions = [
      {
        index_patterns          = ["*"]
        allowed_actions         = ["*"]
        document_level_security = ""
        field_level_security    = []
        masked_fields           = []
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "integration-production-admin"
    error_message = "Production admin role name should match"
  }
}

# ---------------------------------------------------------------------------
# 10. Backup operator role
# ---------------------------------------------------------------------------
run "create_backup_operator_role" {
  module {
    source = "./modules/opensearch_role"
  }

  variables {
    role_name   = "integration-backup-operator"
    description = "Restricted role for snapshot and restore operations"
    cluster_permissions = [
      "cluster:admin/snapshot/*",
      "cluster:admin/repository/*",
      "cluster:monitor/health",
      "cluster:monitor/state"
    ]
    index_permissions = [
      {
        index_patterns          = ["*"]
        allowed_actions         = ["indices:admin/create", "indices:data/read/search", "indices:monitor/stats"]
        document_level_security = ""
        field_level_security    = []
        masked_fields           = []
      }
    ]
  }

  assert {
    condition     = opensearch_role.this.role_name == "integration-backup-operator"
    error_message = "Backup operator role name should match"
  }
}
