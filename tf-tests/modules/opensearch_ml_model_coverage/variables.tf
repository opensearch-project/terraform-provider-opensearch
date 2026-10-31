variable "name_prefix" {
  description = "Prefix used for the model and its dependencies"
  type        = string
}

variable "model_kind" {
  description = "Registration path to test: remote, custom, or pretrained"
  type        = string
  default     = "remote"
}

variable "revision" {
  description = "Configuration revision used to exercise model updates"
  type        = number
  default     = 1
}

variable "include_connector" {
  description = "Whether to create and attach the dependency connector"
  type        = bool
  default     = true
}

variable "guardrail_type" {
  description = "Guardrail type for remote model tests"
  type        = string
  default     = "local_regex"
}

variable "deploy_after_registering" {
  description = "Whether to deploy the model after registration"
  type        = bool
  default     = false
}

variable "wait_for_predict" {
  description = "Whether to wait for predict readiness after deployment"
  type        = bool
  default     = true
}

variable "predict_probe_body" {
  description = "Request body used for the model predict readiness probe"
  type        = string
  default     = "{\"parameters\": {\"inputText\": \"healthcheck\"}}"
}