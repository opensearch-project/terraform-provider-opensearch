provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_pipeline" {
  module {
    source = "./modules/opensearch_ingest_pipeline"
  }

  variables {
    name = "test-pipeline"
    body = jsonencode({
      description = "Test ingest pipeline"
      processors = [
        {
          set = {
            field = "foo"
            value = "bar"
          }
        }
      ]
    })
  }

  assert {
    condition     = opensearch_ingest_pipeline.this.name == "test-pipeline"
    error_message = "Ingest pipeline name should be test-pipeline"
  }
}

run "update_pipeline" {
  module {
    source = "./modules/opensearch_ingest_pipeline"
  }

  variables {
    name = run.create_pipeline.name
    body = jsonencode({
      description = "Updated ingest pipeline"
      processors = [
        {
          set = {
            field = "foo"
            value = "bar"
          }
        },
        {
          set = {
            field = "baz"
            value = "qux"
          }
        }
      ]
    })
  }

  assert {
    condition     = opensearch_ingest_pipeline.this.name == run.create_pipeline.name
    error_message = "Ingest pipeline name should match the previous run"
  }
}

run "create_full_pipeline" {
  module {
    source = "./modules/opensearch_ingest_pipeline"
  }

  variables {
    name = "test-pipeline-full"
    body = jsonencode({
      description = "Full ingest pipeline with multiple processors"
      processors = [
        {
          set = {
            field = "timestamp"
            value = "{{_ingest.timestamp}}"
          }
        },
        {
          remove = {
            field = "temporary"
          }
        },
        {
          date = {
            field   = "log_date"
            formats = ["dd/MMM/yyyy:HH:mm:ss Z"]
          }
        },
        {
          script = {
            lang   = "painless"
            source = "ctx['_source']['level'] = ctx['_source']['level'] != null ? ctx['_source']['level'].toUpperCase() : 'UNKNOWN'"
          }
        }
      ]
    })
  }

  assert {
    condition     = opensearch_ingest_pipeline.this.name == "test-pipeline-full"
    error_message = "Ingest pipeline name should be test-pipeline-full"
  }
}

