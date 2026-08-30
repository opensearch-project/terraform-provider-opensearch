variable "script_id" {
  description = "Script ID"
  type        = string
}

variable "script_source" {
  description = "Script source code"
  type        = string
}

variable "lang" {
  description = "Script language"
  type        = string
  default     = "painless"
}
