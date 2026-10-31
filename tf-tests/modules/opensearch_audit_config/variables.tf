variable "enabled" {
  description = "Whether audit logging is enabled"
  type        = bool
  default     = true
}

variable "enable_rest" {
  description = "Whether to enable REST request logging"
  type        = bool
  default     = true
}

variable "enable_transport" {
  description = "Whether to enable transport request logging"
  type        = bool
  default     = true
}

variable "ignore_users" {
  description = "Set of users to ignore"
  type        = set(string)
  default     = []
}

variable "ignore_requests" {
  description = "Set of requests to ignore"
  type        = set(string)
  default     = []
}

variable "disabled_rest_categories" {
  description = "Set of REST categories to disable"
  type        = set(string)
  default     = []
}

variable "disabled_transport_categories" {
  description = "Set of transport categories to disable"
  type        = set(string)
  default     = []
}

variable "compliance_enabled" {
  description = "Enable compliance"
  type        = bool
  default     = false
}

variable "compliance_write_watched_indices" {
  description = "Set of indices to watch for write compliance"
  type        = set(string)
  default     = []
}
