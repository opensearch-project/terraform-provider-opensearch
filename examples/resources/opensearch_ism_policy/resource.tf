# Create an ISM policy
resource "opensearch_ism_policy" "cleanup" {
  policy_id = "delete_after_15d"
  body      = file("${path.module}/policies/delete_after_15d.json")
}
