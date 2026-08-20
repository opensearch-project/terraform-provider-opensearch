# Import an auto-follow replication rule by "<leader_alias>/<name>": the leader
# alias of a rule is not exposed by the auto-follow API, so it is part of the ID
terraform import opensearch_cross_cluster_replication_rule.movies my-connection-alias/my-replication-rule

# Note: The following field is not returned by the auto-follow stats API and
# will need to be manually added to the Terraform configuration after import:
#
#   - use_roles (only used when creating the rule)
