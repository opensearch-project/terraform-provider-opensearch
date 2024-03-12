# Create an index template
resource "opensearch_index_template" "template_1" {
  name = "template_1"
  body = <<EOF
{
  "index_patterns": [
    "logs-2020-01-*"
  ],
  "template": {
    "aliases": {
      "my_logs": {}
    },
    "settings": {
      "index": {
        "number_of_shards": "2",
        "number_of_replicas": "1"
      }
    },
    "mappings": {
      "properties": {
        "timestamp": {
          "type": "date",
          "format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis"
        },
        "value": {
          "type": "double"
        }
      }
    }
  }
}
EOF
}
