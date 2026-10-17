terraform {
  required_version = ">= 1.11.0"

  required_providers {
    opensearch = {
      source  = "opensearch-project/opensearch"
      version = ">= 2.0.0"
    }
  }
}
