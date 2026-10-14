# =============================================================================
# Core variables
# =============================================================================
variable "opensearch_url" {
  type        = string
  description = "OpenSearch cluster URL. Defaults to the local Docker cluster."
  default     = "http://admin:myStrongPassword123%40456@localhost:9200"
}

variable "sign_aws_requests" {
  type        = bool
  description = "Enable AWS SigV4 request signing for AWS OpenSearch Service."
  default     = false
}

# =============================================================================
# AWS-related variables
#
# AWS credentials required by resources. Set these via TF_VAR_aws_<...> in the
# environment where the make targets are running — see terraform.tfvars.example.
# =============================================================================
variable "aws_access_key" {
  type        = string
  description = "AWS access key."
  default     = ""
  sensitive   = true
}

variable "aws_secret_key" {
  type        = string
  description = "AWS secret key."
  default     = ""
  sensitive   = true
}

variable "aws_session_token" {
  type        = string
  description = "AWS session token (for temporary credentials)."
  default     = ""
  sensitive   = true
}
