# Import a running replication by the name of its follower index
terraform import opensearch_cross_cluster_replication.movies movies

# Note: The following fields are not returned by the replication status API and
# will need to be manually added to the Terraform configuration after import:
#
#   - use_roles (only used when starting or resuming replication)
#   - force_resume (only describes how to resume replication)
#   - settings (the follower index has many settings, only the ones listed in
#     the configuration are managed)
