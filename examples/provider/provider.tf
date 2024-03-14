# Configure the OpenSearch provider
provider "opensearch" {
  url = "http://127.0.0.1:9200"
}

# Create an index template
resource "opensearch_index_template" "template_1" {
  name = "template_1"
  body = <<EOF
{
  "index_patterns": [
    "your-pattern-here-*"
  ],
  "template": {
    "settings": {
      "index": {
        "number_of_shards": "1"
      }
    },
    "mappings": {
      "_source": {
        "enabled": false
      },
      "properties": {
        "host_name": {
          "type": "keyword"
        },
        "created_at": {
          "type": "date",
          "format": "EEE MMM dd HH:mm:ss Z YYYY"
        }
      }
    }
  }
}
EOF
}
