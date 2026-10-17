output "model_group_id" {
  value = opensearch_ml_model_group.this.id
}

output "connector_id" {
  value = opensearch_ml_connector.this.id
}

output "model_id" {
  value = opensearch_ml_model.this.id
}