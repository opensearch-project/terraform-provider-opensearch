resource "opensearch_ml_connector" "this" {
  name          = var.name
  description   = var.description
  version       = var.connector_version
  protocol      = var.protocol
  credential    = var.credential
  parameters    = var.parameters
  access_mode   = var.access_mode
  backend_roles = var.backend_roles

  actions {
    action_type           = "predict"
    method                = "POST"
    url                   = var.action_url
    headers               = var.action_headers
    request_body          = var.request_body
    pre_process_function  = var.pre_process_function
    post_process_function = var.post_process_function
  }

  dynamic "client_config" {
    for_each = var.client_config == null ? [] : [var.client_config]

    content {
      max_connection        = client_config.value.max_connection
      connection_timeout    = client_config.value.connection_timeout
      read_timeout          = client_config.value.read_timeout
      max_retry_times       = client_config.value.max_retry_times
      retry_backoff_policy  = client_config.value.retry_backoff_policy
      retry_backoff_millis  = client_config.value.retry_backoff_millis
      retry_timeout_seconds = client_config.value.retry_timeout_seconds
    }
  }
}