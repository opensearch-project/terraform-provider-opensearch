provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_index_minimal" {

  state_key = "test-index-minimal"

  module {
    source = "./modules/opensearch_index"
  }

  variables {
    name = "test-index-minimal"
  }

  assert {
    condition     = opensearch_index.this.name == "test-index-minimal"
    error_message = "Index name should be test-index-minimal"
  }
}

run "create_index_full_config" {

  state_key = "test-index-full"

  module {
    source = "./modules/opensearch_index"
  }

  variables {
    name               = "test-index-full"
    number_of_shards   = 2
    number_of_replicas = 1
  }

  assert {
    condition     = opensearch_index.this.name == "test-index-full"
    error_message = "Index name should be test-index-full"
  }

  assert {
    condition     = opensearch_index.this.number_of_shards == "2"
    error_message = "Number of shards should be 2"
  }

  assert {
    condition     = opensearch_index.this.number_of_replicas == "1"
    error_message = "Number of replicas should be 1"
  }
}

run "update_index_settings" {

  state_key = "test-index-update"

  module {
    source = "./modules/opensearch_index"
  }

  variables {
    name               = "test-index-update"
    number_of_shards   = 1
    number_of_replicas = 0
  }

  assert {
    condition     = opensearch_index.this.name == "test-index-update"
    error_message = "Index name should be test-index-update"
  }

  assert {
    condition     = opensearch_index.this.number_of_shards == "1"
    error_message = "Number of shards should be 1"
  }

  assert {
    condition     = opensearch_index.this.number_of_replicas == "0"
    error_message = "Number of replicas should be 0"
  }
}

run "update_increase_replicas" {

  state_key = "test-index-update"

  module {
    source = "./modules/opensearch_index"
  }

  variables {
    name               = run.update_index_settings.name
    number_of_shards   = 1
    number_of_replicas = 2
  }

  assert {
    condition     = opensearch_index.this.name == run.update_index_settings.name
    error_message = "Index name should match the previous run"
  }

  assert {
    condition     = opensearch_index.this.number_of_replicas == "2"
    error_message = "Number of replicas should be updated to 2"
  }
}

