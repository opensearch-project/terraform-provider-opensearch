package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchCrossClusterConnection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchCrossClusterConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchCrossClusterConnectionSniff,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterConnectionExists("opensearch_cross_cluster_connection.test"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "id", "terraform-test-connection"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "mode", "sniff"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "seeds.#", "1"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "seeds.0", "127.0.0.1:9300"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "skip_unavailable", "false"),
				),
			},
			{
				Config: testAccOpensearchCrossClusterConnectionSniffUpdated,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterConnectionExists("opensearch_cross_cluster_connection.test"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "skip_unavailable", "true"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "node_connections", "1"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "transport_ping_schedule", "30s"),
				),
			},
			{
				// Removing the optional attributes resets the settings.
				Config: testAccOpensearchCrossClusterConnectionSniff,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchCrossClusterConnectionExists("opensearch_cross_cluster_connection.test"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "skip_unavailable", "false"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "node_connections", "0"),
					resource.TestCheckResourceAttr("opensearch_cross_cluster_connection.test", "transport_ping_schedule", ""),
				),
			},
		},
	})
}

func TestAccOpensearchCrossClusterConnection_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchCrossClusterConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchCrossClusterConnectionSniffUpdated,
			},
			{
				ResourceName:      "opensearch_cross_cluster_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchCrossClusterConnectionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := getResourceIDFromState(s, name)
		if err != nil {
			return err
		}

		settings, err := getCrossClusterConnectionSettings(context.Background(), testAccProvider.Meta().(*ProviderConf), id)
		if err != nil {
			return err
		}
		if len(settings) == 0 {
			return fmt.Errorf("cross-cluster connection %q not found", id)
		}

		return nil
	}
}

func testCheckOpensearchCrossClusterConnectionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_cross_cluster_connection" {
			continue
		}

		settings, err := getCrossClusterConnectionSettings(context.Background(), testAccProvider.Meta().(*ProviderConf), rs.Primary.ID)
		if err != nil {
			return err
		}
		if len(settings) > 0 {
			return fmt.Errorf("cross-cluster connection %q still exists: %v", rs.Primary.ID, settings)
		}
	}

	return nil
}

// The seed is the transport address of the test cluster itself: a cluster can
// register itself as a remote cluster, which keeps this test independent of a
// second cluster.
var testAccOpensearchCrossClusterConnectionSniff = `
resource "opensearch_cross_cluster_connection" "test" {
  name  = "terraform-test-connection"
  seeds = ["127.0.0.1:9300"]
}
`

var testAccOpensearchCrossClusterConnectionSniffUpdated = `
resource "opensearch_cross_cluster_connection" "test" {
  name                    = "terraform-test-connection"
  seeds                   = ["127.0.0.1:9300"]
  node_connections        = 1
  skip_unavailable        = true
  transport_compress      = true
  transport_ping_schedule = "30s"
}
`

func TestCrossClusterConnectionPayload(t *testing.T) {
	t.Run("sniff mode leaves the proxy settings unset", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, resourceOpensearchCrossClusterConnection().Schema, map[string]any{
			"name":             "leader",
			"mode":             "sniff",
			"seeds":            []any{"leader-node:9300"},
			"node_connections": 2,
			"skip_unavailable": true,
		})

		payload := crossClusterConnectionPayload(d, "leader")

		expectedSet := map[string]any{
			"cluster.remote.leader.mode":             "sniff",
			"cluster.remote.leader.node_connections": 2,
			"cluster.remote.leader.skip_unavailable": true,
		}
		for key, expected := range expectedSet {
			if payload[key] != expected {
				t.Errorf("expected %q to be %v, got %v", key, expected, payload[key])
			}
		}

		seeds, ok := payload["cluster.remote.leader.seeds"].([]string)
		if !ok || len(seeds) != 1 || seeds[0] != "leader-node:9300" {
			t.Errorf("expected the seeds to be [leader-node:9300], got %v", payload["cluster.remote.leader.seeds"])
		}

		// Unset attributes are reset explicitly.
		for _, key := range []string{
			"cluster.remote.leader.proxy_address",
			"cluster.remote.leader.proxy_socket_connections",
			"cluster.remote.leader.server_name",
			"cluster.remote.leader.transport.ping_schedule",
		} {
			value, ok := payload[key]
			if !ok {
				t.Errorf("expected %q to be present in the payload", key)
			}
			if value != nil {
				t.Errorf("expected %q to be null, got %v", key, value)
			}
		}
	})

	t.Run("proxy mode leaves the sniff settings unset", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, resourceOpensearchCrossClusterConnection().Schema, map[string]any{
			"name":          "leader",
			"mode":          "proxy",
			"proxy_address": "proxy:9300",
			"server_name":   "leader.example.com",
		})

		payload := crossClusterConnectionPayload(d, "leader")

		if payload["cluster.remote.leader.mode"] != "proxy" {
			t.Errorf("expected the mode to be proxy, got %v", payload["cluster.remote.leader.mode"])
		}
		if payload["cluster.remote.leader.proxy_address"] != "proxy:9300" {
			t.Errorf("expected the proxy address to be proxy:9300, got %v", payload["cluster.remote.leader.proxy_address"])
		}
		if payload["cluster.remote.leader.server_name"] != "leader.example.com" {
			t.Errorf("expected the server name to be leader.example.com, got %v", payload["cluster.remote.leader.server_name"])
		}
		if payload["cluster.remote.leader.seeds"] != nil {
			t.Errorf("expected the seeds to be null, got %v", payload["cluster.remote.leader.seeds"])
		}
		if payload["cluster.remote.leader.node_connections"] != nil {
			t.Errorf("expected the node connections to be null, got %v", payload["cluster.remote.leader.node_connections"])
		}
	})
}

func TestCrossClusterSettingToStringList(t *testing.T) {
	list, err := crossClusterSettingToStringList([]any{"a:9300", "b:9300"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(list) != 2 || list[0] != "a:9300" || list[1] != "b:9300" {
		t.Errorf("unexpected list: %v", list)
	}

	list, err = crossClusterSettingToStringList("a:9300")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(list) != 1 || list[0] != "a:9300" {
		t.Errorf("unexpected list: %v", list)
	}

	if _, err := crossClusterSettingToStringList(42); err == nil {
		t.Error("expected an error for a value that is not a list of strings")
	}
}

func TestValidateCrossClusterConnectionMode(t *testing.T) {
	testCases := []struct {
		name        string
		config      map[string]any
		expectError bool
	}{
		{
			name:   "sniff mode with seeds",
			config: map[string]any{"name": "leader", "seeds": []any{"leader-node:9300"}},
		},
		{
			name:        "sniff mode without seeds",
			config:      map[string]any{"name": "leader"},
			expectError: true,
		},
		{
			name:   "proxy mode with a proxy address",
			config: map[string]any{"name": "leader", "mode": "proxy", "proxy_address": "proxy:9300"},
		},
		{
			name:        "proxy mode without a proxy address",
			config:      map[string]any{"name": "leader", "mode": "proxy"},
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceOpensearchCrossClusterConnection().Schema, testCase.config)
			// TestResourceDataRaw does not apply the schema defaults.
			if _, ok := testCase.config["mode"]; !ok {
				if err := d.Set("mode", "sniff"); err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			err := validateCrossClusterConnectionMode(d)
			if testCase.expectError && err == nil {
				t.Error("expected an error, got none")
			}
			if !testCase.expectError && err != nil {
				t.Errorf("expected no error, got %s", err)
			}
		})
	}
}
