variable "role_name" {
  description = "Name of the role"
  type        = string
}

variable "description" {
  description = "Description"
  type        = string
  default     = ""
}

variable "cluster_permissions" {
  description = "Cluster permissions"
  type        = list(string)
  default     = []
}

variable "index_permissions" {
  description = "Index permissions"
  type = list(object({
    index_patterns          = list(string)
    allowed_actions         = list(string)
    document_level_security = optional(string, "")
    field_level_security    = optional(list(string), [])
    masked_fields           = optional(list(string), [])
  }))
  default = []
}

variable "tenant_permissions" {
  description = "Tenant (dashboard) permissions"
  type = list(object({
    tenant_patterns = list(string)
    allowed_actions = list(string)
  }))
  default = []
}
