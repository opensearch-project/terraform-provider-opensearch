terraform {
  required_providers {
    opensearch = {
      source  = "opensearch-project/opensearch"
      version = ">= 2.0.0"
    }
  }
}

provider "opensearch" {
  url               = var.opensearch_url
  healthcheck       = false
  sign_aws_requests = var.sign_aws_requests
}
