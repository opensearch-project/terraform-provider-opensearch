# Example destination in other terraform plan
# resource "opensearch_destination" "test" {
#   body = <<EOF
# {
#   "name": "my-destination",
#   "type": "slack",
#   "slack": {
#     "url": "http://www.example.com"
#   }
# }
# EOF
# }

data "opensearch_destination" "test" {
  name = "my-destination"
}
