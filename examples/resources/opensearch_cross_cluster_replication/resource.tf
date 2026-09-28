# Replication is started on the follower cluster, so the provider must be
# configured against the follower cluster.
resource "opensearch_cross_cluster_connection" "leader" {
  name  = "my-connection-alias"
  seeds = ["10.0.1.10:9300"]
}

# Minimal replication: creates the follower index "movies" from the index
# "movies" of the leader cluster and keeps it in sync.
resource "opensearch_cross_cluster_replication" "movies" {
  follower_index = "movies"
  leader_alias   = opensearch_cross_cluster_connection.leader.name
  leader_index   = "movies"
}

# With the security plugin enabled, the roles used by the background
# replication tasks on both clusters must be specified.
resource "opensearch_cross_cluster_replication" "logs" {
  follower_index = "logs-follower"
  leader_alias   = opensearch_cross_cluster_connection.leader.name
  leader_index   = "logs"

  use_roles {
    leader_cluster_role   = "cross_cluster_replication_leader_full_access"
    follower_cluster_role = "cross_cluster_replication_follower_full_access"
  }

  # Settings of the follower index, applied with the replication update
  # settings API. Only the settings listed here are managed.
  settings = {
    "index.number_of_replicas" = "1"
  }
}

# Replication can be paused and resumed without recreating the follower index.
# Beware that a replication paused for longer than the retention lease period
# (12 hours by default) can only be resumed with force_resume, which restores
# the follower index from the leader.
resource "opensearch_cross_cluster_replication" "archive" {
  follower_index = "archive-follower"
  leader_alias   = opensearch_cross_cluster_connection.leader.name
  leader_index   = "archive"

  paused       = true
  force_resume = true
}
