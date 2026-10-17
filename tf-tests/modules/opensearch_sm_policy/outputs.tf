output "id" {
  description = "ID of the resource"
  value       = opensearch_sm_policy.this.id
}

output "policy_name" {
  description = "Policy name"
  value       = opensearch_sm_policy.this.policy_name
}
