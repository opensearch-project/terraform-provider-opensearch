# Search for custom Windows rules
data "opensearch_rule" "windows_rules" {
  category = "windows"
}

output "windows_rule_count" {
  value = length(data.opensearch_rule.windows_rules.rules)
}

output "windows_rule_titles" {
  value = [for rule in data.opensearch_rule.windows_rules.rules : rule.title]
}

# Search for pre-packaged high severity rules
data "opensearch_rule" "prepackaged_high" {
  pre_packaged = true
  level        = "high"
}

# Search for experimental network rules
data "opensearch_rule" "experimental_network" {
  category = "network"
  status   = "experimental"
}

# Use rule data to create a resource
data "opensearch_rule" "existing_rule" {
  category = "windows"
}

# Example: Reference a specific rule from the search results
locals {
  first_rule = length(data.opensearch_rule.existing_rule.rules) > 0 ? data.opensearch_rule.existing_rule.rules[0] : null
}

output "first_rule_id" {
  value = local.first_rule != null ? local.first_rule.rule_id : null
}