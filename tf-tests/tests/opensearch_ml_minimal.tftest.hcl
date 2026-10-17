provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "minimal_ml_stack" {
  command   = apply
  state_key = "minimal_ml_stack"

  module {
    source = "./modules/opensearch_ml_minimal"
  }

  variables {
    name_prefix = "tf_test_ml_minimal"
  }

  assert {
    condition     = opensearch_ml_model_group.this.id != ""
    error_message = "Model group should be created"
  }

  assert {
    condition     = opensearch_ml_model_group.this.access_mode == "private"
    error_message = "Model group should default to private access"
  }

  assert {
    condition     = opensearch_ml_connector.this.protocol == "http"
    error_message = "Connector protocol should be HTTP"
  }

  assert {
    condition     = opensearch_ml_connector.this.actions[0].url == "https://api.openai.com/v1/completions"
    error_message = "Connector action URL should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.model_group_id == opensearch_ml_model_group.this.id
    error_message = "Model should reference the created model group"
  }

  assert {
    condition     = opensearch_ml_model.this.connector_id == opensearch_ml_connector.this.id
    error_message = "Model should reference the created connector"
  }

  assert {
    condition     = opensearch_ml_model.this.deploy_after_registering == false
    error_message = "Minimal test model should not deploy"
  }

  assert {
    condition     = opensearch_ml_model.this.is_enabled == true
    error_message = "Model should default to enabled"
  }

  assert {
    condition     = opensearch_ml_model.this.wait_for_predict == true
    error_message = "Model should default to waiting for predict readiness"
  }

  assert {
    condition     = opensearch_ml_model.this.predict_probe_body == "{\"parameters\": {\"inputText\": \"healthcheck\"}}"
    error_message = "Model should use the default predict probe body"
  }
}

run "create_remote_model_full" {
  command   = apply
  state_key = "ml_model_remote_full"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix = "tf_test_ml_model_remote_full"
    model_kind  = "remote"
    revision    = 1
  }

  assert {
    condition     = output.id != ""
    error_message = "Remote model ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model.this.function_name == "REMOTE"
    error_message = "Model should use the REMOTE function"
  }

  assert {
    condition     = opensearch_ml_model.this.model_group_id == output.model_group_id
    error_message = "Remote model should reference its model group"
  }

  assert {
    condition     = opensearch_ml_model.this.connector_id == output.connector_id
    error_message = "Remote model should reference its connector"
  }

  assert {
    condition     = opensearch_ml_model.this.is_enabled == true
    error_message = "Remote model should be enabled"
  }

  assert {
    condition     = opensearch_ml_model.this.wait_for_predict == true
    error_message = "Model should default to waiting for predict readiness"
  }

  assert {
    condition     = opensearch_ml_model.this.rate_limiter[0].limit == 4
    error_message = "Remote model rate limit should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.interface[0].input == "{\"properties\":{\"parameters\":{\"properties\":{\"text\":{\"type\":\"string\"}}}}}"
    error_message = "Remote model interface should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.guardrails[0].type == "local_regex"
    error_message = "Remote model local-regex guardrail type should be retained"
  }
}

run "update_remote_model" {
  command   = apply
  state_key = "ml_model_remote_full"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix = "tf_test_ml_model_remote_full"
    model_kind  = "remote"
    revision    = 2
  }

  assert {
    condition     = output.id == run.create_remote_model_full.id
    error_message = "Updating the remote model should preserve its ID"
  }

  assert {
    condition     = opensearch_ml_model.this.name == "tf_test_ml_model_remote_full_model_2"
    error_message = "Remote model name should be updated"
  }

  assert {
    condition     = opensearch_ml_model.this.is_enabled == false
    error_message = "Remote model should be disabled by the update"
  }

  assert {
    condition     = opensearch_ml_model.this.rate_limiter[0].limit == 10
    error_message = "Remote model rate limit should be updated"
  }

  assert {
    condition     = opensearch_ml_model.this.interface[0].input == "{\"properties\":{\"parameters\":{\"properties\":{\"prompt\":{\"type\":\"string\"}}}}}"
    error_message = "Remote model interface should be updated"
  }
}

run "create_custom_model_full" {
  command   = apply
  state_key = "ml_model_custom_full"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix       = "tf_test_ml_model_custom_full"
    model_kind        = "custom"
    revision          = 1
    include_connector = false
  }

  assert {
    condition     = output.id != ""
    error_message = "Custom model ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model.this.function_name == "TEXT_EMBEDDING"
    error_message = "Custom model should use the TEXT_EMBEDDING function"
  }

  assert {
    condition     = opensearch_ml_model.this.model_config[0].model_type == "bert"
    error_message = "Custom model type should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.model_config[0].embedding_dimension == 384
    error_message = "Custom model embedding dimension should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.model_config[0].pooling_mode == "MEAN"
    error_message = "Custom model pooling mode should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.rate_limiter[0].limit == 4
    error_message = "Custom model rate limit should be retained"
  }
}

run "update_custom_model" {
  command   = apply
  state_key = "ml_model_custom_full"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix       = "tf_test_ml_model_custom_full"
    model_kind        = "custom"
    revision          = 2
    include_connector = false
  }

  assert {
    condition     = output.id == run.create_custom_model_full.id
    error_message = "Updating the custom model should preserve its ID"
  }

  assert {
    condition     = opensearch_ml_model.this.model_config[0].pooling_mode == "MAX"
    error_message = "Custom model pooling mode should be updated"
  }

  assert {
    condition     = opensearch_ml_model.this.model_config[0].all_config == "{\"revision\":2}"
    error_message = "Custom model configuration should be updated"
  }

  assert {
    condition     = opensearch_ml_model.this.model_config[0].normalize_result == true
    error_message = "Custom model normalization should be updated"
  }

  assert {
    condition     = opensearch_ml_model.this.rate_limiter[0].limit == 10
    error_message = "Custom model rate limit should be updated"
  }
}

run "create_pretrained_model" {
  command   = apply
  state_key = "ml_model_pretrained"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix       = "tf_test_ml_model_pretrained"
    model_kind        = "pretrained"
    include_connector = false
  }

  assert {
    condition     = output.id != ""
    error_message = "Pretrained model ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model.this.function_name == "SPARSE_ENCODING"
    error_message = "Pretrained model should use the SPARSE_ENCODING function"
  }

  assert {
    condition     = opensearch_ml_model.this.version == "1.0.0"
    error_message = "Pretrained model version should be retained"
  }

  assert {
    condition     = opensearch_ml_model.this.model_format == "TORCH_SCRIPT"
    error_message = "Pretrained model format should be retained"
  }
}

run "create_model_with_model_guardrails" {
  command   = apply
  state_key = "ml_model_guardrails"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix    = "tf_test_ml_model_guardrails"
    model_kind     = "remote"
    guardrail_type = "model"
  }

  assert {
    condition     = opensearch_ml_model.this.guardrails[0].type == "model"
    error_message = "Remote model should accept model-based guardrails"
  }
}

run "create_remote_model_and_deploy" {
  command   = apply
  state_key = "ml_model_deployed"

  module {
    source = "./modules/opensearch_ml_model_coverage"
  }

  variables {
    name_prefix              = "tf_test_ml_model_deployed"
    model_kind               = "remote"
    deploy_after_registering = true
    wait_for_predict         = false
  }

  assert {
    condition     = output.id != ""
    error_message = "Deployed remote model ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model.this.deploy_after_registering == true
    error_message = "Remote model should deploy after registration"
  }

  assert {
    condition     = opensearch_ml_model.this.wait_for_predict == false
    error_message = "Remote model deployment test should skip external predict"
  }
}
