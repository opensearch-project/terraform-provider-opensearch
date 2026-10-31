variable "username" {
  description = "Username"
  type        = string
}

variable "password" {
  description = "Password"
  type        = string
  sensitive   = true
}

variable "description" {
  description = "Description"
  type        = string
  default     = ""
}

variable "attributes" {
  description = "User attributes"
  type        = map(string)
  default     = {}
}

variable "backend_roles" {
  description = "Backend roles"
  type        = list(string)
  default     = []
}
