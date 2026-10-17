resource "opensearch_sm_policy" "this" {
  policy_name = var.policy_name
  body        = var.body
}
