package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroRolesMapping(t *testing.T) {
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
		CheckDestroy: testAccCheckOpensearchRolesMappingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpenDistroRolesMappingResource(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchRolesMappingExists("opensearch_roles_mapping.test"),
					resource.TestCheckResourceAttr(
						"opensearch_roles_mapping.test",
						"id",
						"readall",
					),
					resource.TestCheckResourceAttr(
						"opensearch_roles_mapping.test",
						"backend_roles.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"opensearch_roles_mapping.test",
						"description",
						randomName,
					),
				),
			},
			{
				Config: testAccOpenDistroRoleMappingResourceUpdated(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchRolesMappingExists("opensearch_roles_mapping.test"),
					resource.TestCheckResourceAttr(
						"opensearch_roles_mapping.test",
						"backend_roles.#",
						"2",
					),
				),
			},
		},
	})
}

func testAccCheckOpensearchRolesMappingDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_roles_mappings_mapping" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchGetOpenDistroRolesMapping(rs.Primary.ID, meta.(*ProviderConf))
		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Role %q still exists", rs.Primary.ID)
	}

	return nil
}
func testCheckOpensearchRolesMappingExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opensearch_roles_mapping" {
				continue
			}

			meta := testAccOpendistroProvider.Meta()
			var err error
			_, err = resourceOpensearchGetOpenDistroRolesMapping(rs.Primary.ID, meta.(*ProviderConf))
			if err != nil {
				return err
			}

			return nil
		}

		return nil
	}
}

func testAccOpenDistroRolesMappingResource(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_roles_mapping" "test" {
  role_name = "readall"
  backend_roles = [
    "active_directory",
  ]

  description = "%s"
}
	`, resourceName)
}

func testAccOpenDistroRoleMappingResourceUpdated(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_roles_mapping" "test" {
  role_name = "readall"
  backend_roles = [
    "active_directory",
    "ldap",
  ]

  description = "%s update"
}
	`, resourceName)
}
