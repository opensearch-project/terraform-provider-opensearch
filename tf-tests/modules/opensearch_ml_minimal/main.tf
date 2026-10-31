resource "opensearch_ml_model_group" "this" {
  name = "${var.name_prefix}_group"
}

resource "opensearch_ml_connector" "this" {
  name        = "${var.name_prefix}_connector"
  description = "Terraform native ML connector test"
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
  name           = "${var.name_prefix}_model"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.this.id
  connector_id   = opensearch_ml_connector.this.id

  deploy_after_registering = false
}