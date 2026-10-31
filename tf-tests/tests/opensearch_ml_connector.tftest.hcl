provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_connector_minimal" {
  command   = apply
  state_key = "ml_connector_minimal"

  module {
    source = "./modules/opensearch_ml_connector"
  }

  variables {
    name = "tf_test_ml_connector_minimal"
  }

  assert {
    condition     = output.id != ""
    error_message = "Connector ID should not be empty"
  }

  assert {
    condition     = output.protocol == "http"
    error_message = "Connector protocol should be HTTP"
  }

  assert {
    condition     = output.action_url == "https://api.openai.com/v1/completions"
    error_message = "Connector action URL should match the configured endpoint"
  }
}

run "create_connector_full" {
  command   = apply
  state_key = "ml_connector_full"

  module {
    source = "./modules/opensearch_ml_connector"
  }

  variables {
    name        = "tf_test_ml_connector_full"
    description = "Full Terraform native ML connector test"
    protocol    = "aws_sigv4"
    credential = {
      access_key = "AKIAIOSFODNN7EXAMPLE"
      secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    }
    parameters = {
      region       = "eu-west-3"
      service_name = "bedrock"
    }
    action_url = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    action_headers = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body          = "{ \"inputText\": \"$${parameters.inputText}\" }"
    pre_process_function  = "connector.pre_process.bedrock.embedding"
    post_process_function = "connector.post_process.bedrock.embedding"
    access_mode           = "restricted"
    backend_roles         = ["ml_full_access"]
    client_config = {
      max_connection        = 10
      connection_timeout    = 10
      read_timeout          = 30
      max_retry_times       = 2
      retry_backoff_policy  = "constant"
      retry_backoff_millis  = 500
      retry_timeout_seconds = 30
    }
  }

  assert {
    condition     = output.id != ""
    error_message = "Connector ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_connector.this.description == "Full Terraform native ML connector test"
    error_message = "Connector description should match the full configuration"
  }

  assert {
    condition     = opensearch_ml_connector.this.version == "1"
    error_message = "Connector version should be retained"
  }

  assert {
    condition     = opensearch_ml_connector.this.access_mode == "restricted"
    error_message = "Connector access mode should be restricted"
  }

  assert {
    condition     = opensearch_ml_connector.this.backend_roles == tolist(["ml_full_access"])
    error_message = "Connector backend roles should be retained"
  }

  assert {
    condition     = opensearch_ml_connector.this.actions[0].pre_process_function == "connector.pre_process.bedrock.embedding"
    error_message = "Connector preprocessing function should be retained"
  }

  assert {
    condition     = opensearch_ml_connector.this.actions[0].post_process_function == "connector.post_process.bedrock.embedding"
    error_message = "Connector postprocessing function should be retained"
  }

  assert {
    condition     = opensearch_ml_connector.this.client_config[0].max_connection == 10
    error_message = "Connector maximum connection count should be retained"
  }

  assert {
    condition     = opensearch_ml_connector.this.client_config[0].retry_backoff_policy == "constant"
    error_message = "Connector retry backoff policy should be retained"
  }

  assert {
    condition     = opensearch_ml_connector.this.client_config[0].retry_timeout_seconds == 30
    error_message = "Connector retry timeout should be retained"
  }
}

run "update_connector_full" {
  command   = apply
  state_key = "ml_connector_full"

  module {
    source = "./modules/opensearch_ml_connector"
  }

  variables {
    name              = "tf_test_ml_connector_full_updated"
    description       = "Updated full Terraform native ML connector test"
    connector_version = "2"
    protocol          = "aws_sigv4"
    credential = {
      access_key = "AKIAIOSFODNN7EXAMPLE"
      secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    }
    parameters = {
      region       = "eu-west-3"
      service_name = "bedrock"
    }
    action_url = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    action_headers = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body          = "{ \"inputText\": \"$${parameters.inputText}\", \"test\": true }"
    pre_process_function  = "connector.pre_process.bedrock.embedding"
    post_process_function = "connector.post_process.bedrock.embedding"
    access_mode           = "restricted"
    backend_roles         = ["ml_full_access"]
    client_config = {
      max_connection        = 20
      connection_timeout    = 20
      read_timeout          = 60
      max_retry_times       = 3
      retry_backoff_policy  = "exponential_equal_jitter"
      retry_backoff_millis  = 600
      retry_timeout_seconds = 60
    }
  }

  assert {
    condition     = output.id == run.create_connector_full.id
    error_message = "Updating the connector should preserve its ID"
  }

  assert {
    condition     = opensearch_ml_connector.this.name == "tf_test_ml_connector_full_updated"
    error_message = "Connector name should be updated"
  }

  assert {
    condition     = opensearch_ml_connector.this.description == "Updated full Terraform native ML connector test"
    error_message = "Connector description should be updated"
  }

  assert {
    condition     = opensearch_ml_connector.this.version == "2"
    error_message = "Connector version should be updated"
  }

  assert {
    condition     = opensearch_ml_connector.this.client_config[0].max_connection == 20
    error_message = "Connector maximum connection count should be updated"
  }

  assert {
    condition     = opensearch_ml_connector.this.client_config[0].retry_backoff_policy == "exponential_equal_jitter"
    error_message = "Connector retry backoff policy should be updated"
  }

  assert {
    condition     = opensearch_ml_connector.this.client_config[0].retry_timeout_seconds == 60
    error_message = "Connector retry timeout should be updated"
  }
}