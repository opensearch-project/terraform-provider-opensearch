resource "opensearch_cluster_settings" "this" {
  cluster_max_shards_per_node                      = var.cluster_max_shards_per_node
  cluster_routing_allocation_total_shards_per_node = var.cluster_routing_allocation_total_shards_per_node
  cluster_routing_allocation_disk_watermark_low    = var.cluster_routing_allocation_disk_watermark_low
  cluster_routing_allocation_disk_watermark_high   = var.cluster_routing_allocation_disk_watermark_high
  action_destructive_requires_name                 = var.action_destructive_requires_name
  indices_breaker_total_limit                      = var.indices_breaker_total_limit
  cluster_info_update_interval                     = var.cluster_info_update_interval
}
