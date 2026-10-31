provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_policy" {
  state_key = "create_policy"

  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = "test-ism-policy"
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
    condition     = opensearch_ism_policy.this.policy_id == "test-ism-policy"
    error_message = "ISM policy ID should be test-ism-policy"
  }
}

run "create_policy_full" {
  state_key = "create_policy_full"

  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = "test-ism-policy-full"
    body = jsonencode({
      policy = {
        description   = "Full ISM policy with multiple states and actions"
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
                conditions = {
                  min_index_age = "3d"
                }
              }
            ]
          },
          {
            name = "warm"
            actions = [
              {
                force_merge = {
                  max_num_segments = 1
                }
              },
              {
                replica_count = {
                  number_of_replicas = 1
                }
              }
            ]
            transitions = [
              {
                state_name = "delete"
                conditions = {
                  min_index_age = "30d"
                }
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
          index_patterns = ["logs-*", "metrics-*"]
          priority       = 200
        }
      }
    })
  }

  assert {
    condition     = opensearch_ism_policy.this.policy_id == "test-ism-policy-full"
    error_message = "ISM policy ID should be test-ism-policy-full"
  }
}

run "read_ism_policy_body" {
  command   = plan
  state_key = "create_policy_full"

  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = "test-ism-policy-full"
    body = jsonencode({
      policy = {
        description   = "Full ISM policy with multiple states and actions"
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
                conditions = {
                  min_index_age = "3d"
                }
              }
            ]
          },
          {
            name = "warm"
            actions = [
              {
                force_merge = {
                  max_num_segments = 1
                }
              },
              {
                replica_count = {
                  number_of_replicas = 1
                }
              }
            ]
            transitions = [
              {
                state_name = "delete"
                conditions = {
                  min_index_age = "30d"
                }
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
          index_patterns = ["logs-*", "metrics-*"]
          priority       = 200
        }
      }
    })
  }

  assert {
    condition     = jsondecode(opensearch_ism_policy.this.body)["policy"]["default_state"] == "hot"
    error_message = "Default state should be hot"
  }

  assert {
    condition     = jsondecode(opensearch_ism_policy.this.body)["policy"]["ism_template"]["priority"] == 200
    error_message = "ISM template priority should be 200"
  }

  assert {
    condition     = jsondecode(opensearch_ism_policy.this.body)["policy"]["states"][0]["name"] == "hot"
    error_message = "First state should be hot"
  }

  assert {
    condition     = jsondecode(opensearch_ism_policy.this.body)["policy"]["states"][2]["name"] == "delete"
    error_message = "Third state should be delete"
  }
}

run "update_policy" {
  state_key = "create_policy"

  module {
    source = "./modules/opensearch_ism_policy"
  }

  variables {
    policy_id = run.create_policy.policy_id
    body = jsonencode({
      policy = {
        description   = "Updated ISM policy"
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
                conditions = {
                  min_index_age = "3d"
                }
              }
            ]
          },
          {
            name = "warm"
            actions = [
              {
                force_merge = {
                  max_num_segments = 1
                }
              }
            ]
            transitions = []
          }
        ]
        ism_template = {
          index_patterns = ["logs-*"]
          priority       = 100
        }
      }
    })
  }

  assert {
    condition     = opensearch_ism_policy.this.policy_id == "test-ism-policy"
    error_message = "ISM policy ID should remain test-ism-policy"
  }
}
