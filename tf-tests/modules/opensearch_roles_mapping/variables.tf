variable "role_name" {
  description = "Name of the role"
  type        = string
}

variable "users" {
  description = "Users to map"
  type        = list(string)
  default     = []
}

variable "backend_roles" {
  description = "Backend roles to map"
  type        = list(string)
  default     = []
}
