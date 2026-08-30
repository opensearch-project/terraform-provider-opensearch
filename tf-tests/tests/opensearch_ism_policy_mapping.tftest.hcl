provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "setup_policy" {
  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = "test-policy"
    body = jsonencode({
      policy = {
        description   = "Test ISM policy"
        default_state = "hot"
        states = [
          {
            name        = "hot"
            actions     = []
            transitions = []
          }
        ]
      }
    })
  }

  assert {
    condition     = opensearch_ism_policy.this.policy_id == "test-policy"
    error_message = "Setup policy ID should be test-policy"
  }
}

run "setup_index" {
  module {
    source = "./modules/opensearch_index"
  }

  variables {
    name = "logs-test"
  }

  assert {
    condition     = opensearch_index.this.name == "logs-test"
    error_message = "Setup index name should be logs-test"
  }
}

run "create_policy_mapping" {
  module {
    source = "./modules/opensearch_ism_policy_mapping"
  }

  variables {
    policy_id = run.setup_policy.policy_id
    indexes   = "logs-*"
  }

  assert {
    condition     = opensearch_ism_policy_mapping.this.policy_id == "test-policy"
    error_message = "Policy ID should be test-policy"
  }
}
