package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	elastic7 "github.com/olivere/elastic/v7"
)

func TestAccOpensearchLogType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchLogTypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchLogTypeResource("test"),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchLogTypeExists("opensearch_log_type.test"),
					resource.TestCheckResourceAttr("opensearch_log_type.test", "name", "test-log-type"),
					resource.TestCheckResourceAttr("opensearch_log_type.test", "description", "Test log type"),
					resource.TestCheckResourceAttr("opensearch_log_type.test", "source", "Custom"),
					resource.TestCheckResourceAttr("opensearch_log_type.test", "category", "Applications"),
				),
			},
			{
				Config: testAccOpensearchLogTypeResourceUpdated("test"),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchLogTypeExists("opensearch_log_type.test"),
					resource.TestCheckResourceAttr("opensearch_log_type.test", "description", "Updated test log type"),
				),
			},
		},
	})
}

func TestAccOpensearchLogType_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchLogTypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchLogTypeResource("test"),
			},
			{
				ResourceName:      "opensearch_log_type.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchLogTypeExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No log type ID is set")
		}

		meta := testAccProvider.Meta()
		_, err := resourceOpensearchGetLogType(rs.Primary.ID, meta)
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchLogTypeDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_log_type" {
			continue
		}

		meta := testAccProvider.Meta()
		_, err := resourceOpensearchGetLogType(rs.Primary.ID, meta)
		if err != nil {
			if elastic7.IsNotFound(err) {
				return nil
			}
			return err
		}

		return fmt.Errorf("Log type %q still exists", rs.Primary.ID)
	}

	return nil
}

func testAccOpensearchLogTypeResource(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_log_type" "%s" {
  name        = "test-log-type"
  description = "Test log type"
  source      = "Custom"
  category    = "Applications"
}
`, resourceName)
}

func testAccOpensearchLogTypeResourceUpdated(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_log_type" "%s" {
  name        = "test-log-type"
  description = "Updated test log type"
  source      = "Custom"
  category    = "Security"
}
`, resourceName)
}
