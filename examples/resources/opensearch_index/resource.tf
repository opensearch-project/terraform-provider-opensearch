# Create a simple index
resource "opensearch_index" "test-simple-index" {
  name               = "terraform-test"
  number_of_shards   = "1"
  number_of_replicas = "1"
  mappings           = <<EOF
{
  "properties": {
    "name": {
      "type": "text"
    }
  }
}
EOF
}
