package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroActionGroup(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	randomName := "test" + acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testAccCheckOpensearchActionGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpenDistroActionGroupResource(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchActionGroupExists("opensearch_action_group.test"),
					resource.TestCheckResourceAttr(
						"opensearch_action_group.test",
						"id",
						randomName,
					),
					resource.TestCheckResourceAttr(
						"opensearch_action_group.test",
						"allowed_actions.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"opensearch_action_group.test",
						"description",
						"test",
					),
				),
			},
			{
				Config: testAccOpenDistroActionGroupResourceUpdated(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchActionGroupExists("opensearch_action_group.test"),
					resource.TestCheckResourceAttr(
						"opensearch_action_group.test",
						"allowed_actions.#",
						"3",
					),
					resource.TestCheckResourceAttr(
						"opensearch_action_group.test",
						"description",
						"test updated",
					),
				),
			},
			{
				ResourceName:      "opensearch_action_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckOpensearchActionGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_action_group" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		if _, err := resourceOpensearchGetOpenDistroActionGroup(rs.Primary.ID, meta.(*ProviderConf)); err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Action group %q still exists", rs.Primary.ID)
	}

	return nil
}

func testCheckOpensearchActionGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opensearch_action_group" {
				continue
			}

			meta := testAccOpendistroProvider.Meta()

			if _, err := resourceOpensearchGetOpenDistroActionGroup(rs.Primary.ID, meta.(*ProviderConf)); err != nil {
				return err
			}

			return nil
		}

		return fmt.Errorf("Action group %q not found", name)
	}
}

func testAccOpenDistroActionGroupResource(name string) string {
	return fmt.Sprintf(`
	resource "opensearch_action_group" "test" {
		action_group_name = "%s"
		description       = "test"
		type              = "index"
		allowed_actions   = ["indices:data/read/search", "indices:data/read/get"]
	}
	`, name)
}

func testAccOpenDistroActionGroupResourceUpdated(name string) string {
	return fmt.Sprintf(`
	resource "opensearch_action_group" "test" {
		action_group_name = "%s"
		description       = "test updated"
		type              = "index"
		allowed_actions   = [
			"indices:data/read/search",
			"indices:data/read/get",
			"indices:data/read/mget*",
		]
	}
	`, name)
}

// The PUT body is what the security plugin persists, so pin its shape: the
// server-managed flags (reserved/hidden/static) must never be sent, and an
// unset optional field must be omitted rather than sent as an empty string.
func TestActionGroupBodyMarshal(t *testing.T) {
	tests := []struct {
		name     string
		body     ActionGroupBody
		expected string
	}{
		{
			name: "full body",
			body: ActionGroupBody{
				AllowedActions: []string{"indices:data/read/search"},
				Type:           "index",
				Description:    "test",
			},
			expected: `{"allowed_actions":["indices:data/read/search"],"type":"index","description":"test"}`,
		},
		{
			name: "optional fields omitted when unset",
			body: ActionGroupBody{
				AllowedActions: []string{"cluster:monitor/main"},
			},
			expected: `{"allowed_actions":["cluster:monitor/main"]}`,
		},
		{
			name: "server-managed flags are never sent",
			body: ActionGroupBody{
				AllowedActions: []string{"cluster:monitor/main"},
				Reserved:       false,
				Hidden:         false,
				Static:         false,
			},
			expected: `{"allowed_actions":["cluster:monitor/main"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

// A GET returns the group keyed by its name, alongside the server-managed
// flags. Unmarshalling must pick up those flags so a future change can refuse
// to modify a reserved or static group.
func TestActionGroupBodyUnmarshal(t *testing.T) {
	raw := []byte(`{
		"my_group": {
			"reserved": true,
			"hidden": false,
			"static": true,
			"type": "index",
			"description": "built in",
			"allowed_actions": ["indices:data/read/search", "indices:data/read/get"]
		}
	}`)

	var groups map[string]ActionGroupBody
	if err := json.Unmarshal(raw, &groups); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	got, ok := groups["my_group"]
	if !ok {
		t.Fatal("expected my_group to be present")
	}
	if len(got.AllowedActions) != 2 {
		t.Errorf("got %d allowed_actions, want 2", len(got.AllowedActions))
	}
	if !got.Reserved {
		t.Error("expected reserved to be true")
	}
	if !got.Static {
		t.Error("expected static to be true")
	}
	if got.Hidden {
		t.Error("expected hidden to be false")
	}
	if got.Type != "index" {
		t.Errorf("got type %q, want %q", got.Type, "index")
	}
}
