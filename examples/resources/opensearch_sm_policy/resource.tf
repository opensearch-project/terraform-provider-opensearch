# Create an SM policy
resource "opensearch_sm_policy" "snapshot_to_s3" {
  policy_name = "snapshot_to_s3"
  body        = file("${path.module}/policies/snapshot_to_s3.json")
}
