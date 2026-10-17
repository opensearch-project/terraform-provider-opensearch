provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_channel" {
  state_key = "create_channel"

  module {
    source = "./modules/opensearch_channel_configuration"
  }

  variables {
    body = jsonencode({
      config = {
        name        = "Test Channel"
        config_type = "slack"
        is_enabled  = true
        slack = {
          url = "https://hooks.slack.com/services/test"
        }
      }
    })
  }

  assert {
    condition     = opensearch_channel_configuration.this.id != ""
    error_message = "Channel configuration ID should not be empty"
  }
}

run "create_channel_full" {
  state_key = "create_channel_full"

  module {
    source = "./modules/opensearch_channel_configuration"
  }

  variables {
    body = jsonencode({
      config = {
        name        = "Full Test Channel"
        description = "Full notification channel with metadata"
        config_type = "slack"
        is_enabled  = true
        slack = {
          url = "https://hooks.slack.com/services/T00000000/B00000000/full"
        }
      }
    })
  }

  assert {
    condition     = opensearch_channel_configuration.this.id != ""
    error_message = "Channel configuration ID should not be empty for full config"
  }
}

run "read_channel_body" {
  command   = plan
  state_key = "create_channel_full"

  module {
    source = "./modules/opensearch_channel_configuration"
  }

  variables {
    body = jsonencode({
      config = {
        name        = "Full Test Channel"
        description = "Full notification channel with metadata"
        config_type = "slack"
        is_enabled  = true
        slack = {
          url = "https://hooks.slack.com/services/T00000000/B00000000/full"
        }
      }
    })
  }

  assert {
    condition     = jsondecode(opensearch_channel_configuration.this.body)["config"]["name"] == "Full Test Channel"
    error_message = "Channel name should be Full Test Channel"
  }

  assert {
    condition     = jsondecode(opensearch_channel_configuration.this.body)["config"]["config_type"] == "slack"
    error_message = "Channel config_type should be slack"
  }

  assert {
    condition     = jsondecode(opensearch_channel_configuration.this.body)["config"]["is_enabled"] == true
    error_message = "Channel is_enabled should be true"
  }

  assert {
    condition     = jsondecode(opensearch_channel_configuration.this.body)["config"]["slack"]["url"] == "https://hooks.slack.com/services/T00000000/B00000000/full"
    error_message = "Slack URL should match"
  }
}

run "update_channel" {
  state_key = "create_channel"

  module {
    source = "./modules/opensearch_channel_configuration"
  }

  variables {
    body = jsonencode({
      config = {
        name        = "Updated Test Channel"
        config_type = "slack"
        is_enabled  = true
        slack = {
          url = "https://hooks.slack.com/services/updated"
        }
      }
    })
  }

  assert {
    condition     = opensearch_channel_configuration.this.id != ""
    error_message = "Channel configuration ID should not be empty after update"
  }
}
