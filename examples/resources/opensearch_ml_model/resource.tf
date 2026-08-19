# Minimal ML Model Group
resource "opensearch_ml_model_group" "minimal" {
  name = "minimal"
}

# Minimal ML Model - OpenSearch-provided pretrained text embedding model
resource "opensearch_ml_model" "os_provided_pretrained_text_embedding_minimal" {
  name           = "huggingface/sentence-transformers/msmarco-distilbert-base-tas-b"
  version        = "1.0.3"
  model_format   = "TORCH_SCRIPT"
  model_group_id = opensearch_ml_model_group.minimal.id
}

# ML Model - OpenSearch-provided pretrained text embedding model with optional attributes
resource "opensearch_ml_model" "os_provided_pretrained_text_embedding_with_optional" {
  name           = "huggingface/sentence-transformers/msmarco-distilbert-base-tas-b"
  description    = "ML Model for OpenSearch-provided pretrained text embedding model"
  version        = "1.0.3"
  model_format   = "TORCH_SCRIPT"
  model_group_id = opensearch_ml_model_group.minimal.id
}

# Minimal ML Model - OpenSearch-provided pretrained sparse encoding model
resource "opensearch_ml_model" "os_provided_pretrained_sparse_encoding_minimal" {
  name           = "amazon/neural-sparse/opensearch-neural-sparse-encoding-doc-v3-distill"
  version        = "1.0.0"
  model_format   = "TORCH_SCRIPT"
  function_name  = "SPARSE_ENCODING"
  model_group_id = opensearch_ml_model_group.minimal.id
}

# Minimal ML Model - Custom model
resource "opensearch_ml_model" "custom_minimal" {
  name                     = "custom_minimal"
  model_group_id           = opensearch_ml_model_group.minimal.id
  version                  = "1.0.1"
  model_format             = "TORCH_SCRIPT"
  function_name            = "TEXT_EMBEDDING"
  url                      = "https://artifacts.opensearch.org/models/ml-models/huggingface/sentence-transformers/all-MiniLM-L6-v2/1.0.1/torch_script/sentence-transformers_all-MiniLM-L6-v2-1.0.1-torch_script.zip"
  model_content_hash_value = "c15f0d2e62d872be5b5bc6c84d2e0f4921541e29fefbef51d59cc10a8ae30e0f"

  model_config {
    model_type          = "bert"
    embedding_dimension = 384
    framework_type      = "sentence_transformers"
  }
}

# ML Model - Custom model with optional attributes
resource "opensearch_ml_model" "custom_with_optional" {
  name                     = "custom_with_optional"
  description              = "ML Model for custom model with optional attributes"
  version                  = "1.0.1"
  model_format             = "TORCH_SCRIPT"
  function_name            = "TEXT_EMBEDDING"
  model_content_hash_value = "c15f0d2e62d872be5b5bc6c84d2e0f4921541e29fefbef51d59cc10a8ae30e0f"
  url                      = "https://artifacts.opensearch.org/models/ml-models/huggingface/sentence-transformers/all-MiniLM-L6-v2/1.0.1/torch_script/sentence-transformers_all-MiniLM-L6-v2-1.0.1-torch_script.zip"
  model_group_id           = opensearch_ml_model_group.minimal.id
  is_enabled               = true

  model_config {
    model_type          = "bert"
    embedding_dimension = 384
    framework_type      = "sentence_transformers"
    pooling_mode        = "mean"
    normalize_result    = true
    all_config          = "{\"_name_or_path\":\"nreimers/MiniLM-L6-H384-uncased\",\"architectures\":[\"BertModel\"],\"attention_probs_dropout_prob\":0.1,\"gradient_checkpointing\":false,\"hidden_act\":\"gelu\",\"hidden_dropout_prob\":0.1,\"hidden_size\":384,\"initializer_range\":0.02,\"intermediate_size\":1536,\"layer_norm_eps\":1e-12,\"max_position_embeddings\":512,\"model_type\":\"bert\",\"num_attention_heads\":12,\"num_hidden_layers\":6,\"pad_token_id\":0,\"position_embedding_type\":\"absolute\",\"transformers_version\":\"4.8.2\",\"type_vocab_size\":2,\"use_cache\":true,\"vocab_size\":30522}"
    additional_config {
      space_type = "l2"
    }
  }

  rate_limiter {
    limit = 4
    unit  = "SECONDS"
  }

  interface {
    input = "{\"properties\":{\"parameters\":{\"properties\":{\"messages\":{\"type\":\"string\",\"description\":\"Test description field\"}}}}}"
  }
}

# Minimal ML Connector with 'aws_sigv4' protocol
resource "opensearch_ml_connector" "minimal" {
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
      content-type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
}

# Minimal ML Model - Third-party-hosted model
resource "opensearch_ml_model" "third_party_minimal" {
  name           = "third_party_minimal"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.minimal.id
  connector_id   = opensearch_ml_connector.minimal.id
}

# ML Model - Third-party-hosted model with optional attributes and local-regex-based guardrails
resource "opensearch_ml_model" "third_party_with_optional" {
  name           = "third_party_with_optional"
  description    = "ML Model for third-party model with optional attributes"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.minimal.id
  connector_id   = opensearch_ml_connector.minimal.id
  is_enabled     = true

  rate_limiter {
    limit = 4
    unit  = "SECONDS"
  }

  interface {
    input = "{\"properties\":{\"parameters\":{\"properties\":{\"messages\":{\"type\":\"string\",\"description\":\"Test description field\"}}}}}"
  }

  guardrails {
    type = "local_regex"
    input_guardrail {
      stop_words {
        index_name    = "stop_words_input"
        source_fields = ["messages"]
      }
      regex = [".*kill.*"]
    }
    output_guardrail {
      regex = [".*forbidden.*"]
    }
  }
}

# ML Model - Third-party-hosted model with model-based guardrails
resource "opensearch_ml_model" "third_party_with_model_guardrails" {
  name           = "third_party_with_model_guardrails"
  description    = "ML Model for third-party model with model-based guardrails"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.minimal.id
  connector_id   = opensearch_ml_connector.minimal.id
  is_enabled     = true

  guardrails {
    type                      = "model"
    model_id                  = "guardrail_model_id"
    response_validation_regex = "^\\d{4}-\\d{2}-\\d{2}$"
  }
}
