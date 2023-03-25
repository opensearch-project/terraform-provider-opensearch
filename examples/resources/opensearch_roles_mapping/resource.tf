# Create a role mapping
resource "opensearch_roles_mapping" "mapper" {
  role_name   = "logs_writer"
  description = "Mapping AWS IAM roles to ES role"
  backend_roles = [
    "arn:aws:iam::123456789012:role/lambda-call-opensearch",
    "arn:aws:iam::123456789012:role/run-containers",
  ]
}
