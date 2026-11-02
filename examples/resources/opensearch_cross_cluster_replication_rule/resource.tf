# Auto-follow rules are created on the follower cluster, so the provider must
# be configured against the follower cluster.
resource "opensearch_cross_cluster_connection" "leader" {
  name  = "my-connection-alias"
  seeds = ["10.0.1.10:9300"]
}

# Every index of the leader cluster matching the pattern is replicated, both
# the ones that already exist and the ones created later.
resource "opensearch_cross_cluster_replication_rule" "movies" {
  name         = "my-replication-rule"
  leader_alias = opensearch_cross_cluster_connection.leader.name
  pattern      = "movies*"
}

# With the security plugin enabled, the roles used by the background
# replication tasks on both clusters must be specified.
resource "opensearch_cross_cluster_replication_rule" "logs" {
  name         = "logs-replication-rule"
  leader_alias = opensearch_cross_cluster_connection.leader.name
  pattern      = "logs-*"

  use_roles {
    leader_cluster_role   = "cross_cluster_replication_leader_full_access"
    follower_cluster_role = "cross_cluster_replication_follower_full_access"
  }
}
