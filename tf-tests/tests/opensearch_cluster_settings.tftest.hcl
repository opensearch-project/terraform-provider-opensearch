provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_cluster_settings" {
  state_key = "create_cluster_settings"

  module {
    source = "./modules/opensearch_cluster_settings"
  }

  variables {
    cluster_max_shards_per_node = 100
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_max_shards_per_node == 100
    error_message = "Max shards per node should be 100"
  }
}

run "create_cluster_settings_full" {
  state_key = "create_cluster_settings_full"

  module {
    source = "./modules/opensearch_cluster_settings"
  }

  variables {
    cluster_max_shards_per_node                      = 150
    cluster_routing_allocation_total_shards_per_node = 2000
    cluster_routing_allocation_disk_watermark_low    = "80%"
    cluster_routing_allocation_disk_watermark_high   = "85%"
    action_destructive_requires_name                 = "true"
    indices_breaker_total_limit                      = "95%"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_max_shards_per_node == 150
    error_message = "Max shards per node should be 150"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_routing_allocation_total_shards_per_node == 2000
    error_message = "Total shards per node should be 2000"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_routing_allocation_disk_watermark_low == "80%"
    error_message = "Low watermark should be 80%"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_routing_allocation_disk_watermark_high == "85%"
    error_message = "High watermark should be 85%"
  }

  assert {
    condition     = opensearch_cluster_settings.this.action_destructive_requires_name == true
    error_message = "Destructive requires name should be true"
  }

  assert {
    condition     = opensearch_cluster_settings.this.indices_breaker_total_limit == "95%"
    error_message = "Indices breaker total limit should be 95%"
  }
}

run "update_cluster_settings" {
  state_key = "update_cluster_settings"

  module {
    source = "./modules/opensearch_cluster_settings"
  }

  variables {
    cluster_max_shards_per_node                    = 200
    cluster_routing_allocation_disk_watermark_low  = "85%"
    cluster_routing_allocation_disk_watermark_high = "90%"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_max_shards_per_node == 200
    error_message = "Max shards per node should be updated to 200"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_routing_allocation_disk_watermark_low == "85%"
    error_message = "Low watermark should be 85%"
  }

  assert {
    condition     = opensearch_cluster_settings.this.cluster_routing_allocation_disk_watermark_high == "90%"
    error_message = "High watermark should be 90%"
  }
}
