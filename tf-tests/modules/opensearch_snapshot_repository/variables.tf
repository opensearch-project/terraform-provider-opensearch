variable "name" {
  description = "Name of the repository"
  type        = string
}

variable "type" {
  description = "Type of the repository"
  type        = string
}

variable "settings" {
  description = "Repository settings"
  type        = map(string)
  default     = {}
}
