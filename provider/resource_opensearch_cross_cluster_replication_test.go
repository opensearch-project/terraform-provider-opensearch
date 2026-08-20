package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Cross-cluster replication requires a second (leader) cluster, which the
// default test cluster of `make test` does not provide. The tests below are
// skipped unless both variables are set, see `make test-ccr`:
//
//   - OPENSEARCH_CCR_LEADER_URL: the HTTP URL of the leader cluster, used to
//     create the replicated indexes.
//   - OPENSEARCH_CCR_LEADER_SEED: the transport address (`host:port`) of the
//     leader cluster, as reachable from the follower cluster.
const (
	ccrLeaderURLEnvVar  = "OPENSEARCH_CCR_LEADER_URL"
	ccrLeaderSeedEnvVar = "OPENSEARCH_CCR_LEADER_SEED"
)

// Runs before the test case is even built, so it skips rather than fails when
// the environment is not there — including OPENSEARCH_URL, which the test case
// PreCheck can no longer report on its own once this helper has run.
func testAccCrossClusterReplicationPreCheck(t *testing.T) (leaderURL string, leaderSeed string) {
	leaderURL = os.Getenv(ccrLeaderURLEnvVar)
	leaderSeed = os.Getenv(ccrLeaderSeedEnvVar)
	if leaderURL == "" || leaderSeed == "" || os.Getenv("OPENSEARCH_URL") == "" {
		t.Skipf("OPENSEARCH_URL, %s and %s must be set to run the cross-cluster replication tests, see `make test-ccr`", ccrLeaderURLEnvVar, ccrLeaderSeedEnvVar)
	}

	return leaderURL, leaderSeed
}

// Performs a raw request against one of the clusters, e.g. to manage the
// leader indexes, which the provider is not connected to. Missing resources
// are not an error, so that cleanups are idempotent.
func testAccClusterRequest(method string, clusterURL string, path string, body string) error {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, strings.TrimSuffix(clusterURL, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("%s %s failed (status %d): %s", method, path, res.StatusCode, responseBody)
	}

	return nil
}

func TestAccOpensearchCrossClusterReplication(t *testing.T) {
	leaderURL, leaderSeed := testAccCrossClusterReplicationPreCheck(t)

	leaderIndex := "terraform-test-ccr-leader"
	followerIndex := "terraform-test-ccr-follower"

	if err := testAccClusterRequest("PUT", leaderURL, "/"+leaderIndex, `{"settings":{"index.number_of_shards":1,"index.number_of_replicas":0}}`); err != nil {
		t.Fatal(err)
	}

	// Stopping replication leaves the follower index behind, and the plugin
	// refuses to replicate onto an existing index, so it has to be deleted
	// before and after the test to keep the runs reproducible.
	followerURL := os.Getenv("OPENSEARCH_URL")
	if err := testAccClusterRequest("DELETE", followerURL, "/"+followerIndex, ""); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := testAccClusterRequest("DELETE", leaderURL, "/"+leaderIndex, ""); err != nil {
			t.Logf("failed to delete the leader index: %s", err)
		}
		if err := testAccClusterRequest("DELETE", followerURL, "/"+followerIndex, ""); err != nil {
			t.Logf("failed to delete the follower index: %s", err)
		}
	})

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchCrossClusterReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchCrossClusterReplicationConfig(leaderSeed, leaderIndex, followerIndex, false, ""),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterReplicationStatus("opensearch_cross_cluster_replication.test", replicationStatusSyncing),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "id", followerIndex),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "leader_index", leaderIndex),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "leader_alias", "terraform-test-ccr"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "paused", "false"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "status", replicationStatusSyncing),
				),
			},
			{
				// Updating the settings of the follower index.
				Config: testAccOpensearchCrossClusterReplicationConfig(leaderSeed, leaderIndex, followerIndex, false, `"index.number_of_replicas" = "0"`),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterReplicationStatus("opensearch_cross_cluster_replication.test", replicationStatusSyncing),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "settings.index.number_of_replicas", "0"),
				),
			},
			{
				// Pausing replication.
				Config: testAccOpensearchCrossClusterReplicationConfig(leaderSeed, leaderIndex, followerIndex, true, `"index.number_of_replicas" = "0"`),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterReplicationStatus("opensearch_cross_cluster_replication.test", replicationStatusPaused),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "paused", "true"),
				),
			},
			{
				// Resuming replication.
				Config: testAccOpensearchCrossClusterReplicationConfig(leaderSeed, leaderIndex, followerIndex, false, `"index.number_of_replicas" = "0"`),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterReplicationStatus("opensearch_cross_cluster_replication.test", replicationStatusSyncing),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication.test", "paused", "false"),
				),
			},
			{
				ResourceName:      "opensearch_cross_cluster_replication.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The roles are not exposed by the replication status API, and
				// force_resume only describes how to resume.
				ImportStateVerifyIgnore: []string{"use_roles", "force_resume", "settings"},
			},
		},
	})
}

func testCheckOpensearchCrossClusterReplicationStatus(name string, expectedStatus string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := getResourceIDFromState(s, name)
		if err != nil {
			return err
		}

		status, err := getCrossClusterReplicationStatus(context.Background(), testAccProvider.Meta().(*ProviderConf), id)
		if err != nil {
			return err
		}

		if state, _ := status["status"].(string); !strings.EqualFold(state, expectedStatus) {
			return fmt.Errorf("replication of index %q is in status %q, expected %q", id, state, expectedStatus)
		}

		return nil
	}
}

func testCheckOpensearchCrossClusterReplicationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_cross_cluster_replication" {
			continue
		}

		status, err := getCrossClusterReplicationStatus(context.Background(), testAccProvider.Meta().(*ProviderConf), rs.Primary.ID)
		if err != nil {
			// The follower index is gone altogether.
			return nil
		}

		if state, _ := status["status"].(string); !strings.EqualFold(state, replicationStatusNotInProgress) {
			return fmt.Errorf("replication of index %q still exists with status %q", rs.Primary.ID, state)
		}
	}

	return nil
}

func testAccOpensearchCrossClusterReplicationConfig(leaderSeed string, leaderIndex string, followerIndex string, paused bool, settings string) string {
	settingsBlock := ""
	if settings != "" {
		settingsBlock = fmt.Sprintf("\n  settings = {\n    %s\n  }\n", settings)
	}

	return fmt.Sprintf(`
resource "opensearch_cross_cluster_connection" "test" {
  name  = "terraform-test-ccr"
  seeds = ["%s"]
}

resource "opensearch_cross_cluster_replication" "test" {
  follower_index = "%s"
  leader_alias   = opensearch_cross_cluster_connection.test.name
  leader_index   = "%s"
  paused         = %t
%s
  use_roles {
    leader_cluster_role   = "all_access"
    follower_cluster_role = "all_access"
  }
}
`, leaderSeed, followerIndex, leaderIndex, paused, settingsBlock)
}

func TestExpandCrossClusterReplicationUseRoles(t *testing.T) {
	if roles := expandCrossClusterReplicationUseRoles([]any{}); roles != nil {
		t.Errorf("expected no roles for an empty configuration, got %v", roles)
	}

	if roles := expandCrossClusterReplicationUseRoles([]any{nil}); roles != nil {
		t.Errorf("expected no roles for an empty block, got %v", roles)
	}

	roles := expandCrossClusterReplicationUseRoles([]any{
		map[string]any{
			"leader_cluster_role":   "cross_cluster_replication_leader_full_access",
			"follower_cluster_role": "cross_cluster_replication_follower_full_access",
		},
	})
	if roles["leader_cluster_role"] != "cross_cluster_replication_leader_full_access" {
		t.Errorf("unexpected leader cluster role: %v", roles["leader_cluster_role"])
	}
	if roles["follower_cluster_role"] != "cross_cluster_replication_follower_full_access" {
		t.Errorf("unexpected follower cluster role: %v", roles["follower_cluster_role"])
	}
}
