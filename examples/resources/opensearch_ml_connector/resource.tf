# Notes on 'aws_sigv4' credentials
# --------------------------------
# The 'aws_sigv4' protocol supports two credential forms:
#
#   1. IAM Role assumption — credential = { roleArn = "..." }
#      Works only on AWS-managed OpenSearch deployments (e.g. Amazon
#      OpenSearch Service) where the cluster has an attached IAM identity
#      that is allowed to call sts:AssumeRole on the target role.
#      On self-managed clusters (local Docker, on-prem, etc.) this form is
#      rejected by ML Commons with "Missing credential".
#   2. Direct credentials — credential = { access_key, secret_key, session_token }
#      Works on any OpenSearch deployment. The role must be assumed in advance
#      and the resulting temporary credentials must be piped to Terraform.
#
# The examples below use the 'roleArn' form for brevity; see
# 'minimal_self_managed_aws_sigv4' for the direct-credential alternative.

# Minimal ML Connector with 'aws_sigv4' protocol (IAM Role — AWS-managed OpenSearch)
resource "opensearch_ml_connector" "minimal_aws_sigv4" {
  name        = "minimal_aws_sigv4"
  description = "Minimal ML Connector with only the mandatory attributes connecting to AWS Bedrock with an IAM Role"
  version     = "1"
  protocol    = "aws_sigv4"
  credential = {
    roleArn = "<ARN of IAM Role with permissions to access the AWS Bedrock model>"
  }
  parameters = {
    region       = "eu-west-3"
    service_name = "bedrock"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers     = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
}

# Minimal ML Connector with 'aws_sigv4' protocol using direct credentials
# (for self-managed clusters that can't perform sts:AssumeRole themselves)
resource "opensearch_ml_connector" "minimal_self_managed_aws_sigv4" {
  name        = "self_managed_aws_sigv4"
  description = "AWS Bedrock connector using pre-assumed temporary credentials"
  version     = "1"
  protocol    = "aws_sigv4"
  credential = {
    access_key    = "<AWS access key>"
    secret_key    = "<AWS secret key>"
    session_token = "<AWS session token>"
  }
  parameters = {
    region       = "eu-west-3"
    service_name = "bedrock"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers     = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
}

# Minimal ML Connector with 'http' protocol
resource "opensearch_ml_connector" "minimal_http" {
  name        = "minimal_http"
  description = "Minimal ML Connector with only the mandatory attributes connecting to OpenAI with an API Key"
  version     = "1"
  protocol    = "http"
  credential = {
    openAIKey = "<OpenAI API Key>"
  }
  parameters = {
    endpoint    = "api.openai.com"
    max_tokens  = 7
    temperature = 0
    model       = "gpt-4.1"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://$${parameters.endpoint}/v1/completions"
    headers     = {
      Authorization = "Bearer $${credential.openAIKey}"
    }
    request_body = "{ \"model\": \"$${parameters.model}\", \"prompt\": \"$${parameters.prompt}\", \"max_tokens\": \"$${parameters.max_tokens}\", \"temperature\": \"$${parameters.temperature}\" }"
  }
}

# ML Connector with Pre and Post processing functions
resource "opensearch_ml_connector" "with_pre_post_processing" {
  name        = "with_pre_post_processing"
  description = "ML Connector pre and post processing functions"
  version     = "1"
  protocol    = "aws_sigv4"
  credential = {
    roleArn = "<ARN of IAM Role with permissions to access the AWS Bedrock model>"
  }
  parameters = {
    region       = "eu-west-3"
    service_name = "bedrock"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
    pre_process_function  = "connector.pre_process.bedrock.embedding"
    post_process_function = "connector.post_process.bedrock.embedding"
  }
}

# ML Connector with 'access_mode' and 'backend_roles'
resource "opensearch_ml_connector" "with_access_mode_and_backend_roles" {
  name        = "with_access_mode_and_backend_roles"
  description = "ML Connector with 'access_mode' set to 'restricted' and specific 'backend_roles'"
  version     = "1"
  protocol    = "aws_sigv4"
  credential = {
    roleArn = "<ARN of IAM Role with permissions to access the AWS Bedrock model>"
  }
  parameters = {
    region       = "eu-west-3"
    service_name = "bedrock"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers     = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  access_mode   = "restricted"
  backend_roles = ["ml_full_access"]
}

# ML Connector with Client Config
resource "opensearch_ml_connector" "with_client_config" {
  name        = "with_client_config"
  description = "ML Connector with all 'client_config' attributes"
  version     = "1"
  protocol    = "aws_sigv4"
  credential = {
    roleArn = "<ARN of IAM Role with permissions to access the AWS Bedrock model>"
  }
  parameters = {
    region       = "eu-west-3"
    service_name = "bedrock"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers     = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  client_config {
    max_connection        = 10
    connection_timeout    = 10
    read_timeout          = 10
    retry_backoff_policy  = "constant"
    max_retry_times       = 2
    retry_backoff_millis  = 500
    retry_timeout_seconds = 30
  }
}
