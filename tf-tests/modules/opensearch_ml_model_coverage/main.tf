locals {
  model_name = var.model_kind == "pretrained" ? "amazon/neural-sparse/opensearch-neural-sparse-encoding-doc-v2-mini" : "${var.name_prefix}_model_${var.revision}"
  function_name = var.model_kind == "custom" ? "TEXT_EMBEDDING" : (
    var.model_kind == "pretrained" ? "SPARSE_ENCODING" : "REMOTE"
  )
}

resource "opensearch_ml_model_group" "this" {
  name = "${var.name_prefix}_group"
}

resource "opensearch_ml_connector" "this" {
  count       = var.include_connector ? 1 : 0
  name        = "${var.name_prefix}_connector"
  description = "Model test dependency connector"
  version     = "1"
  protocol    = "http"

  credential = {
    openAIKey = "not-a-real-key"
  }

  parameters = {
    endpoint = "api.openai.com"
    model    = "gpt-4.1"
  }

  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://api.openai.com/v1/completions"
    headers = {
      Authorization = "Bearer $${credential.openAIKey}"
    }
    request_body = "{\"model\": \"$${parameters.model}\", \"prompt\": \"Terraform test\", \"max_tokens\": 1}"
  }
}

resource "opensearch_ml_model" "this" {
  name           = local.model_name
  description    = "Terraform native ${var.model_kind} model test revision ${var.revision}"
  function_name  = local.function_name
  model_group_id = opensearch_ml_model_group.this.id
  connector_id   = var.model_kind == "remote" && var.include_connector ? opensearch_ml_connector.this[0].id : null
  is_enabled     = var.model_kind == "remote" && var.revision == 2 ? false : true

  version = var.model_kind == "custom" ? "1.0.2" : (
    var.model_kind == "pretrained" ? "1.0.0" : null
  )
  model_format = var.model_kind == "remote" ? null : "TORCH_SCRIPT"

  url                      = var.model_kind == "custom" ? "https://artifacts.opensearch.org/models/ml-models/huggingface/sentence-transformers/paraphrase-MiniLM-L3-v2/1.0.2/torch_script/sentence-transformers_paraphrase-MiniLM-L3-v2-1.0.2-torch_script.zip" : null
  model_content_hash_value = var.model_kind == "custom" ? "843d3246ed04369593f1c54f2be92dc9878d60d5610b89617c585619e9f162d0" : null

  deploy_after_registering = var.deploy_after_registering
  wait_for_predict         = var.wait_for_predict
  predict_probe_body       = var.predict_probe_body

  dynamic "model_config" {
    for_each = var.model_kind == "custom" ? [1] : []

    content {
      model_type          = "bert"
      embedding_dimension = 384
      framework_type      = "sentence_transformers"
      pooling_mode        = var.revision == 1 ? "MEAN" : "MAX"
      all_config          = jsonencode({ revision = var.revision })
      normalize_result    = var.revision != 1

      additional_config {
        space_type = var.revision == 1 ? "l2" : "cosinesimil"
      }
    }
  }

  dynamic "rate_limiter" {
    for_each = var.model_kind != "pretrained" ? [1] : []

    content {
      limit = var.revision == 1 ? 4 : 10
      unit  = var.revision == 1 ? "SECONDS" : "MINUTES"
    }
  }

  dynamic "interface" {
    for_each = var.model_kind != "pretrained" ? [1] : []

    content {
      input = var.revision == 1 ? "{\"properties\":{\"parameters\":{\"properties\":{\"text\":{\"type\":\"string\"}}}}}" : "{\"properties\":{\"parameters\":{\"properties\":{\"prompt\":{\"type\":\"string\"}}}}}"
    }
  }

  dynamic "guardrails" {
    for_each = var.model_kind == "remote" ? [1] : []

    content {
      type                      = var.guardrail_type
      model_id                  = var.guardrail_type == "model" ? "test_guardrail_model_id" : null
      response_validation_regex = var.guardrail_type == "model" ? "^[0-9]{4}$" : null

      dynamic "input_guardrail" {
        for_each = var.guardrail_type == "local_regex" ? [1] : []

        content {
          regex = [var.revision == 1 ? ".*blocked.*" : ".*restricted.*"]
        }
      }

      dynamic "output_guardrail" {
        for_each = var.guardrail_type == "local_regex" ? [1] : []

        content {
          regex = [var.revision == 1 ? ".*secret.*" : ".*private.*"]
        }
      }
    }
  }
}