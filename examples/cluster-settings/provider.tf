# Configure the OpenSearch provider
terraform {
  required_providers {
    opensearch = {
      source = "registry.terraform.io/opensearch-project/opensearch"
    }
  }
}

provider "opensearch" {
  url = "http://127.0.0.1:9200"
  username = "admin"
  password = "myStrongPassword123@456"
}

resource "opensearch_cluster_settings" "persistent" {
  cluster_max_shards_per_node = 10
  cluster_search_request_slowlog_level            = "WARN"
  cluster_search_request_slowlog_threshold_warn   = "10s"
  cluster_search_request_slowlog_threshold_info   = "5s"
  cluster_search_request_slowlog_threshold_debug  = "2s"
  cluster_search_request_slowlog_threshold_trace  = "100ms"
  cluster_routing_allocation_awareness_attributes = "zone"
  cluster_routing_allocation_awareness_force_zone_values = ["zoneA", "zoneB"]
}
