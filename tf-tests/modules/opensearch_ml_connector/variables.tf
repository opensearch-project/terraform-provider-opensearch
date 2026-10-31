variable "name" {
  description = "Connector name"
  type        = string
}

variable "description" {
  description = "Connector description"
  type        = string
  default     = "Terraform native ML connector resource test"
}

variable "connector_version" {
  description = "Connector version"
  type        = string
  default     = "1"
}

variable "protocol" {
  description = "Connector protocol"
  type        = string
  default     = "http"
}

variable "credential" {
  description = "Connector credentials"
  type        = map(string)
  default     = { openAIKey = "not-a-real-key" }
  sensitive   = true
}

variable "parameters" {
  description = "Connector parameters"
  type        = map(string)
  default = {
    endpoint = "api.openai.com"
    model    = "gpt-4.1"
  }
}

variable "action_url" {
  description = "Connector action URL"
  type        = string
  default     = "https://api.openai.com/v1/completions"
}

variable "action_headers" {
  description = "Connector action headers"
  type        = map(string)
  default = {
    Authorization = "Bearer $${credential.openAIKey}"
  }
}

variable "request_body" {
  description = "Connector action request body"
  type        = string
  default     = "{\"model\": \"$${parameters.model}\", \"prompt\": \"Terraform test\", \"max_tokens\": 1}"
}

variable "pre_process_function" {
  description = "Optional action preprocessing function"
  type        = string
  default     = null
}

variable "post_process_function" {
  description = "Optional action postprocessing function"
  type        = string
  default     = null
}

variable "access_mode" {
  description = "Optional connector access mode"
  type        = string
  default     = null
}

variable "backend_roles" {
  description = "Optional connector backend roles"
  type        = list(string)
  default     = null
}

variable "client_config" {
  description = "Optional connector client configuration"
  type = object({
    max_connection        = number
    connection_timeout    = number
    read_timeout          = number
    max_retry_times       = number
    retry_backoff_policy  = string
    retry_backoff_millis  = number
    retry_timeout_seconds = number
  })
  default = null
}