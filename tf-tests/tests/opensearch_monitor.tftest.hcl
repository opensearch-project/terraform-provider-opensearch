provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_monitor" {
  state_key = "create_monitor"

  module {
    source = "./modules/opensearch_monitor"
  }

  variables {
    body = jsonencode({
      type    = "monitor"
      name    = "Test Monitor"
      enabled = true
      schedule = {
        period = {
          interval = 1
          unit     = "MINUTES"
        }
      }
      inputs   = []
      triggers = []
    })
  }

  assert {
    condition     = opensearch_monitor.this.id != ""
    error_message = "Monitor ID should not be empty"
  }
}

run "create_monitor_with_trigger" {
  state_key = "create_monitor_with_trigger"

  module {
    source = "./modules/opensearch_monitor"
  }

  variables {
    body = jsonencode({
      type    = "monitor"
      name    = "Full Test Monitor"
      enabled = true
      schedule = {
        period = {
          interval = 10
          unit     = "MINUTES"
        }
      }
      inputs = [
        {
          search = {
            indices = ["*"]
            query = {
              query = {
                match_all = {}
              }
              size = 0
            }
          }
        }
      ]
      triggers = [
        {
          name     = "test-trigger"
          severity = "1"
          condition = {
            script = {
              source = "ctx.results[0].hits.total.value > 0"
              lang   = "painless"
            }
          }
          actions = []
        }
      ]
    })
  }

  assert {
    condition     = opensearch_monitor.this.id != ""
    error_message = "Full monitor ID should not be empty"
  }
}

run "read_monitor_body" {
  command   = plan
  state_key = "create_monitor_with_trigger"

  module {
    source = "./modules/opensearch_monitor"
  }

  variables {
    body = jsonencode({
      type    = "monitor"
      name    = "Full Test Monitor"
      enabled = true
      schedule = {
        period = {
          interval = 10
          unit     = "MINUTES"
        }
      }
      inputs = [
        {
          search = {
            indices = ["*"]
            query = {
              query = {
                match_all = {}
              }
              size = 0
            }
          }
        }
      ]
      triggers = [
        {
          name     = "test-trigger"
          severity = "1"
          condition = {
            script = {
              source = "ctx.results[0].hits.total.value > 0"
              lang   = "painless"
            }
          }
          actions = []
        }
      ]
    })
  }

  assert {
    condition     = jsondecode(opensearch_monitor.this.body)["type"] == "monitor"
    error_message = "Monitor type should be monitor"
  }

  assert {
    condition     = jsondecode(opensearch_monitor.this.body)["name"] == "Full Test Monitor"
    error_message = "Monitor name should be Full Test Monitor"
  }

  assert {
    condition     = jsondecode(opensearch_monitor.this.body)["enabled"] == true
    error_message = "Monitor should be enabled"
  }

  assert {
    condition     = jsondecode(opensearch_monitor.this.body)["schedule"]["period"]["interval"] == 10
    error_message = "Schedule interval should be 10"
  }

  assert {
    condition     = jsondecode(opensearch_monitor.this.body)["inputs"][0]["search"]["indices"][0] == "*"
    error_message = "Input indices should be asterisk"
  }

  assert {
    condition     = jsondecode(opensearch_monitor.this.body)["triggers"][0]["name"] == "test-trigger"
    error_message = "Trigger name should be test-trigger"
  }
}

run "update_monitor" {
  state_key = "update_monitor"

  module {
    source = "./modules/opensearch_monitor"
  }

  variables {
    body = jsonencode({
      type    = "monitor"
      name    = "Updated Monitor"
      enabled = true
      schedule = {
        period = {
          interval = 5
          unit     = "MINUTES"
        }
      }
      inputs   = []
      triggers = []
    })
  }

  assert {
    condition     = opensearch_monitor.this.id != ""
    error_message = "Monitor ID should not be empty after update"
  }
}
