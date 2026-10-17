resource "opensearch_ism_policy" "this" {
  policy_id = var.policy_id
  body      = var.body
}
