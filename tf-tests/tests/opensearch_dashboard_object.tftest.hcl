provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_object" {
  module {
    source = "./modules/opensearch_dashboard_object"
  }

  variables {
    body = jsonencode([
      {
        _id = "dashboard:test"
        _source = {
          type = "dashboard"
          dashboard = {
            title = "Test Dashboard"
          }
        }
      }
    ])
  }

  assert {
    condition     = opensearch_dashboard_object.this.id != ""
    error_message = "Dashboard object ID should not be empty"
  }
}

run "update_object" {
  module {
    source = "./modules/opensearch_dashboard_object"
  }

  variables {
    body = jsonencode([
      {
        _id = "dashboard:test"
        _source = {
          type = "dashboard"
          dashboard = {
            title = "Updated Dashboard"
          }
        }
      }
    ])
  }

  assert {
    condition     = opensearch_dashboard_object.this.id != ""
    error_message = "Dashboard object ID should not be empty after update"
  }
}

run "create_multiple_objects" {
  module {
    source = "./modules/opensearch_dashboard_object"
  }

  variables {
    body = jsonencode([
      {
        _id = "index-pattern:test-*"
        _source = {
          type = "index-pattern"
          "index-pattern" = {
            title         = "test-*"
            timeFieldName = "@timestamp"
          }
        }
      },
      {
        _id = "visualization:test-viz"
        _source = {
          type = "visualization"
          visualization = {
            title = "Test Visualization"
          }
        }
      },
      {
        _id = "dashboard:test-dashboard"
        _source = {
          type = "dashboard"
          dashboard = {
            title      = "Full Test Dashboard"
            panelsJSON = "[]"
          }
        }
      }
    ])
  }

  assert {
    condition     = opensearch_dashboard_object.this.id != ""
    error_message = "Dashboard object ID should not be empty for multi-object"
  }
}
