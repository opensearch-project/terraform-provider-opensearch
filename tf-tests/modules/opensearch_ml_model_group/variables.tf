variable "name" {
  description = "Model group name"
  type        = string
}

variable "description" {
  description = "Optional model group description"
  type        = string
  default     = null
}

variable "access_mode" {
  description = "Optional model group access mode"
  type        = string
  default     = null
}

variable "backend_roles" {
  description = "Optional backend roles for a restricted model group"
  type        = list(string)
  default     = null
}

variable "add_all_backend_roles" {
  description = "Whether to add all owner backend roles"
  type        = bool
  default     = null
}