package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchCrossClusterReplicationRule(t *testing.T) {
	_, leaderSeed := testAccCrossClusterReplicationPreCheck(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchCrossClusterReplicationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchCrossClusterReplicationRuleConfig(leaderSeed, "terraform-test-ccr-rule-*"),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterReplicationRuleExists("opensearch_cross_cluster_replication_rule.test"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication_rule.test", "id", "terraform-test-ccr/terraform-test-rule"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication_rule.test", "name", "terraform-test-rule"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication_rule.test", "leader_alias", "terraform-test-ccr"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication_rule.test", "pattern", "terraform-test-ccr-rule-*"),
				),
			},
			{
				// The plugin rejects updates of an existing rule, changing the
				// pattern recreates it.
				Config: testAccOpensearchCrossClusterReplicationRuleConfig(leaderSeed, "terraform-test-ccr-other-*"),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterReplicationRuleExists("opensearch_cross_cluster_replication_rule.test"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication_rule.test", "id", "terraform-test-ccr/terraform-test-rule"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_replication_rule.test", "pattern", "terraform-test-ccr-other-*"),
				),
			},
			{
				ResourceName:      "opensearch_cross_cluster_replication_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The roles are not exposed by the auto-follow stats API.
				ImportStateVerifyIgnore: []string{"use_roles"},
			},
		},
	})
}

func testCheckOpensearchCrossClusterReplicationRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := getResourceIDFromState(s, name)
		if err != nil {
			return err
		}

		leaderAlias, ruleName, err := parseCrossClusterReplicationRuleID(id)
		if err != nil {
			return err
		}

		rule, err := getCrossClusterReplicationRule(context.Background(), testAccProvider.Meta().(*ProviderConf), leaderAlias, ruleName)
		if err != nil {
			return err
		}
		if rule == nil {
			return fmt.Errorf("replication rule %q not found", id)
		}

		return nil
	}
}

func testCheckOpensearchCrossClusterReplicationRuleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_cross_cluster_replication_rule" {
			continue
		}

		leaderAlias, ruleName, err := parseCrossClusterReplicationRuleID(rs.Primary.ID)
		if err != nil {
			return err
		}

		rule, err := getCrossClusterReplicationRule(context.Background(), testAccProvider.Meta().(*ProviderConf), leaderAlias, ruleName)
		if err != nil {
			return err
		}
		if rule != nil {
			return fmt.Errorf("replication rule %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccOpensearchCrossClusterReplicationRuleConfig(leaderSeed string, pattern string) string {
	return fmt.Sprintf(`
resource "opensearch_cross_cluster_connection" "test" {
  name  = "terraform-test-ccr"
  seeds = ["%s"]
}

resource "opensearch_cross_cluster_replication_rule" "test" {
  name         = "terraform-test-rule"
  leader_alias = opensearch_cross_cluster_connection.test.name
  pattern      = "%s"

  use_roles {
    leader_cluster_role   = "all_access"
    follower_cluster_role = "all_access"
  }
}
`, leaderSeed, pattern)
}

func TestParseCrossClusterReplicationRuleID(t *testing.T) {
	testCases := []struct {
		id                  string
		expectedLeaderAlias string
		expectedName        string
		expectError         bool
	}{
		{id: "leader/my-rule", expectedLeaderAlias: "leader", expectedName: "my-rule"},
		{id: "leader/my/rule", expectedLeaderAlias: "leader", expectedName: "my/rule"},
		{id: "my-rule", expectError: true},
		{id: "/my-rule", expectError: true},
		{id: "leader/", expectError: true},
		{id: "", expectError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			leaderAlias, name, err := parseCrossClusterReplicationRuleID(testCase.id)
			if testCase.expectError {
				if err == nil {
					t.Errorf("expected an error for the ID %q, got none", testCase.id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if leaderAlias != testCase.expectedLeaderAlias {
				t.Errorf("expected the leader alias %q, got %q", testCase.expectedLeaderAlias, leaderAlias)
			}
			if name != testCase.expectedName {
				t.Errorf("expected the name %q, got %q", testCase.expectedName, name)
			}
		})
	}
}

func TestCrossClusterReplicationRuleID(t *testing.T) {
	id := crossClusterReplicationRuleID("leader", "my-rule")
	if id != "leader/my-rule" {
		t.Errorf("expected the ID leader/my-rule, got %q", id)
	}

	leaderAlias, name, err := parseCrossClusterReplicationRuleID(id)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if leaderAlias != "leader" || name != "my-rule" {
		t.Errorf("the ID did not round-trip: %q, %q", leaderAlias, name)
	}
}
