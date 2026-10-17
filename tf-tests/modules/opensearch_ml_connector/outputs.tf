output "id" {
  value = opensearch_ml_connector.this.id
}

output "protocol" {
  value = opensearch_ml_connector.this.protocol
}

output "action_url" {
  value = opensearch_ml_connector.this.actions[0].url
}