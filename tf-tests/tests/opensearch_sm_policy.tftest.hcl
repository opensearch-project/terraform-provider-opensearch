provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "setup_repo" {
  state_key = "setup_repo"

  module {
    source = "./modules/opensearch_snapshot_repository"
  }

  variables {
    name = "test-repo"
    type = "fs"
    settings = {
      location = "/tmp/snapshots"
    }
  }

  assert {
    condition     = opensearch_snapshot_repository.this.name == "test-repo"
    error_message = "Repository name should be test-repo"
  }
}

run "create_sm_policy" {
  state_key = "create_sm_policy"

  module {
    source = "./modules/opensearch_sm_policy"
  }

  variables {
    policy_name = "test-sm-policy"
    body = jsonencode({
      description = "Test SM policy"
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
        indices    = "*"
        repository = run.setup_repo.name
      }
    })
  }

  assert {
    condition     = opensearch_sm_policy.this.policy_name == "test-sm-policy"
    error_message = "SM policy name should be test-sm-policy"
  }
}

run "create_sm_policy_full" {
  state_key = "create_sm_policy_full"

  module {
    source = "./modules/opensearch_sm_policy"
  }

  variables {
    policy_name = "test-sm-policy-full"
    body = jsonencode({
      description = "Full SM policy with detailed snapshot configuration"
      creation = {
        schedule = {
          cron = {
            expression = "0 0 * * 0"
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
          max_count = 50
          min_count = 10
          min_age   = "30d"
        }
      }
      snapshot_config = {
        indices              = "logs-*,metrics-*"
        repository           = run.setup_repo.name
        ignore_unavailable   = true
        include_global_state = false
        partial              = true
      }
    })
  }

  assert {
    condition     = opensearch_sm_policy.this.policy_name == "test-sm-policy-full"
    error_message = "SM policy name should be test-sm-policy-full"
  }
}

run "read_sm_policy_body" {
  command   = plan
  state_key = "create_sm_policy_full"

  module {
    source = "./modules/opensearch_sm_policy"
  }

  variables {
    policy_name = "test-sm-policy-full"
    body = jsonencode({
      description = "Full SM policy with detailed snapshot configuration"
      creation = {
        schedule = {
          cron = {
            expression = "0 0 * * 0"
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
          max_count = 50
          min_count = 10
          min_age   = "30d"
        }
      }
      snapshot_config = {
        indices              = "logs-*,metrics-*"
        repository           = run.setup_repo.name
        ignore_unavailable   = true
        include_global_state = false
        partial              = true
      }
    })
  }

  assert {
    condition     = jsondecode(opensearch_sm_policy.this.body)["description"] == "Full SM policy with detailed snapshot configuration"
    error_message = "Description should match"
  }

  assert {
    condition     = jsondecode(opensearch_sm_policy.this.body)["creation"]["schedule"]["cron"]["expression"] == "0 0 * * 0"
    error_message = "Creation cron expression should be 0 0 * * 0"
  }

  assert {
    condition     = jsondecode(opensearch_sm_policy.this.body)["deletion"]["condition"]["max_count"] == 50
    error_message = "Deletion max_count should be 50"
  }

  assert {
    condition     = jsondecode(opensearch_sm_policy.this.body)["snapshot_config"]["indices"] == "logs-*,metrics-*"
    error_message = "Snapshot indices should match"
  }
}

run "update_sm_policy" {
  state_key = "create_sm_policy"

  module {
    source = "./modules/opensearch_sm_policy"
  }

  variables {
    policy_name = run.create_sm_policy.policy_name
    body = jsonencode({
      description = "Updated SM policy"
      creation = {
        schedule = {
          cron = {
            expression = "0 3 * * *"
            timezone   = "UTC"
          }
        }
      }
      deletion = {
        schedule = {
          cron = {
            expression = "0 4 * * *"
            timezone   = "UTC"
          }
        }
        condition = {
          max_count = 20
          min_count = 3
          min_age   = "14d"
        }
      }
      snapshot_config = {
        indices    = "*"
        repository = run.setup_repo.name
      }
    })
  }

  assert {
    condition     = opensearch_sm_policy.this.policy_name == "test-sm-policy"
    error_message = "SM policy name should remain test-sm-policy"
  }
}
