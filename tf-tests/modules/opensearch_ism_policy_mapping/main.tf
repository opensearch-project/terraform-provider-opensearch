resource "opensearch_ism_policy_mapping" "this" {
  policy_id = var.policy_id
  indexes   = var.indexes
}
