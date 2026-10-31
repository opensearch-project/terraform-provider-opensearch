variable "cluster_max_shards_per_node" {
  description = "Maximum shards per node"
  type        = number
  default     = null
}

variable "cluster_routing_allocation_total_shards_per_node" {
  description = "Total shards per node"
  type        = number
  default     = null
}

variable "cluster_routing_allocation_disk_watermark_low" {
  description = "Low watermark for disk usage"
  type        = string
  default     = null
}

variable "cluster_routing_allocation_disk_watermark_high" {
  description = "High watermark for disk usage"
  type        = string
  default     = null
}

variable "action_destructive_requires_name" {
  description = "Require explicit index names for destructive operations"
  type        = string
  default     = null
}

variable "indices_breaker_total_limit" {
  description = "Total circuit breaker limit"
  type        = string
  default     = null
}

variable "cluster_info_update_interval" {
  description = "Cluster info update interval"
  type        = string
  default     = null
}
