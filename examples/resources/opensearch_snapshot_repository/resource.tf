# Create a snapshot repository
resource "opensearch_snapshot_repository" "repo" {
  name = "es-index-backups"
  type = "s3"
  settings = {
    bucket   = "es-index-backups"
    region   = "us-east-1"
    role_arn = "arn:aws:iam::123456789012:role/MyRole"
  }
}
