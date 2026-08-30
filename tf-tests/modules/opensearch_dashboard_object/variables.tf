variable "name" {
  description = "Name of the resource"
  type        = string
  default     = ""
}

variable "body" {
  description = "Resource configuration (JSON)"
  type        = string
  default     = "{}"
}
