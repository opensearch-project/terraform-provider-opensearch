# The cross-cluster connection is created on the follower cluster: cross-cluster
# replication follows a "pull" model, so the provider must be configured against
# the follower cluster.

# Sniff mode (the default): the follower connects to the seed nodes and
# discovers the other nodes of the leader cluster. The seeds are transport
# addresses (port 9300 by default), not REST addresses.
resource "opensearch_cross_cluster_connection" "leader" {
  name  = "my-connection-alias"
  seeds = ["10.0.1.10:9300", "10.0.1.11:9300"]
}

# Proxy mode: all the connections go through a single address, which is useful
# when the nodes of the leader cluster have no publicly reachable publish
# address (e.g. behind a load balancer).
resource "opensearch_cross_cluster_connection" "leader_behind_proxy" {
  name          = "my-proxied-connection-alias"
  mode          = "proxy"
  proxy_address = "leader-proxy.example.com:9300"
  server_name   = "leader.example.com"

  proxy_socket_connections = 18
  skip_unavailable         = true
}

# The alias of the connection is the leader_alias of the replication resources.
resource "opensearch_cross_cluster_replication" "movies" {
  follower_index = "movies"
  leader_alias   = opensearch_cross_cluster_connection.leader.name
  leader_index   = "movies"
}
