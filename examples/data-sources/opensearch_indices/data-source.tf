# All the indexes of the cluster
data "opensearch_indices" "all" {}

# Stop the replication of the indexes created by an auto-follow rule. Destroying
# the rule only stops *new* indexes from being replicated.
data "opensearch_indices" "movies" {
  pattern = "movies-*"
}

locals {
  # The follower indexes of a running (syncing or paused) replication.
  followers = {
    for index in data.opensearch_indices.movies.indices : index.name => index
    if index.leader_alias != ""
  }
}

# 1. Import the replications, the plan contains no change. `use_roles` is not
#    readable back: setting it would recreate the replications.
import {
  for_each = local.followers

  to = opensearch_cross_cluster_replication.movies[each.key]
  id = each.key
}

resource "opensearch_cross_cluster_replication" "movies" {
  for_each = local.followers

  follower_index = each.key
  leader_alias   = each.value.leader_alias
  leader_index   = each.value.leader_index
}

# 2. Remove the import and resource blocks: destroying the replications stops
#    them, the follower indexes become regular, writable indexes. They are no
#    longer listed in `local.followers`.
