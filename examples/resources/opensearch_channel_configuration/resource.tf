# Create a channel configuration
resource "opensearch_channel_configuration" "configuration_1" {
  body = <<EOF
{
  "config_id": "configuration_1",
  "config": {
    "name": "name",
    "description" : "description",
    "config_type" : "slack",
    "is_enabled" : true,
    "slack": {
      "url": "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
    }
  }
}
EOF
}
