package provider

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchClusterSettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchClusterSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchClusterSettings,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchClusterSettingInState("opensearch_cluster_settings.global"),
					testCheckOpensearchClusterSettingExists("action.auto_create_index"),
				),
			},
		},
	})
}

func testCheckOpensearchClusterSettingInState(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("cluster ID not set")
		}

		return nil
	}
}

func testCheckOpensearchClusterSettingExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meta := testAccProvider.Meta()
		settings, err := resourceOpensearchClusterSettingsGet(meta)
		if err != nil {
			return err
		}

		persistentSettings := settings["persistent"].(map[string]interface{})
		_, ok := persistentSettings[name]
		if !ok {
			return fmt.Errorf("%s not found in settings, found %+v", name, persistentSettings)
		}

		return nil
	}
}

func checkOpensearchClusterSettingsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_cluster_settings" {
			continue
		}

		meta := testAccProvider.Meta()
		settings, err := resourceOpensearchClusterSettingsGet(meta)
		if err != nil {
			return err
		}

		persistentSettings := settings["persistent"].(map[string]interface{})
		if len(persistentSettings) != 0 {
			log.Printf("[INFO] checkOpensearchClusterSettingsDestroy: %+v", persistentSettings)
			return fmt.Errorf("%d cluster settings still exist", len(persistentSettings))
		}

		return nil
	}

	return nil
}

var testAccOpensearchClusterSettings = `
resource "opensearch_cluster_settings" "global" {
  cluster_max_shards_per_node = 10
  cluster_routing_allocation_enable = "all"
  action_auto_create_index    = "my-index-000001,index10,-index1*,+ind*,-.aws_cold_catalog*,+*"
}
`
