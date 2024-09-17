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

## Index with aliases
resource "opensearch_index" "index" {
  name = "sample"
  aliases = jsonencode(
   {
    "log": {
      "is_write_index": true
    }
  }
  )
  number_of_replicas = "1"
  number_of_shards = "1"
  mappings = jsonencode({
    "properties": {
      "age": {
        "type": "integer"
      }
    }
  })
}
