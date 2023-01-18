package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	elastic7 "github.com/olivere/elastic/v7"
)

func TestAccOpensearchOpenDistroUser(t *testing.T) {
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
		CheckDestroy: testAccCheckOpensearchUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpenDistroUserResource(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchUserExists("opensearch_user.test"),
					testCheckOpensearchUserConnects("opensearch_user.test"),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"id",
						randomName,
					),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"backend_roles.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"attributes.some_attribute",
						"alpha",
					),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"description",
						"test",
					),
				),
			},
			{
				Config: testAccOpenDistroUserResourceUpdated(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchUserExists("opensearch_user.test"),
					testCheckOpensearchUserConnects("opensearch_user.test"),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"backend_roles.#",
						"2",
					),
				),
			},
			{
				Config: testAccOpenDistroUserResourceMinimal(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchUserExists("opensearch_user.test"),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"backend_roles.#",
						"0",
					),
				),
			},
			{
				Config: testAccOpenDistroUserResourceHash(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchUserExists("opensearch_user.test"),
					resource.TestCheckResourceAttr(
						"opensearch_user.test",
						"id",
						randomName,
					),
				),
			},
		},
	})
}

func TestAccOpensearchOpenDistroUserMultiple(t *testing.T) {
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
		CheckDestroy: testAccCheckOpensearchUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpenDistroUserMultiple(randomName),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchUserExists("opensearch_user.testuser1"),
				),
			},
		},
	})
}

func testAccCheckOpensearchUserDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_user" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchGetOpenDistroUser(rs.Primary.ID, meta.(*ProviderConf))
		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("User %q still exists", rs.Primary.ID)
	}

	return nil
}

func testCheckOpensearchUserExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opensearch_user" {
				continue
			}

			meta := testAccOpendistroProvider.Meta()

			var err error
			_, err = resourceOpensearchGetOpenDistroUser(rs.Primary.ID, meta.(*ProviderConf))
			if err != nil {
				return err
			}

			return nil
		}

		return nil
	}
}

func testCheckOpensearchUserConnects(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opensearch_user" {
				continue
			}

			username := rs.Primary.Attributes["username"]
			password := rs.Primary.Attributes["password"]

			var err error
			if err != nil {
				return err
			}
			var client *elastic7.Client
			client, err = elastic7.NewClient(
				elastic7.SetURL(os.Getenv("OPENSEARCH_URL")),
				elastic7.SetBasicAuth(username, password))

			if err == nil {
				_, err = client.ClusterHealth().Do(context.TODO())
			}

			if err != nil {
				return err
			}

			return nil
		}

		return nil
	}
}

func testAccOpenDistroUserResource(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_user" "test" {
  username      = "%s"
  password      = "passw0rd"
  description   = "test"
  backend_roles = ["some_role"]

  attributes = {
    some_attribute = "alpha"
  }
}
	`, resourceName)
}

func testAccOpenDistroUserResourceHash(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_user" "test" {
  username      = "%s"
  password_hash = "$2a$04$jQcEXpODnTFoGDuA7DPdSevA84CuH/7MOYkb80M3XZIrH76YMWS9G"
}
	`, resourceName)
}

func testAccOpenDistroUserResourceUpdated(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_user" "test" {
  username      = "%s"
  password      = "passw0rd"
  description   = "test"
  backend_roles = ["some_role", "monitor_role"]

  attributes = {
    some_attribute  = "alpha"
    other_attribute = "beta"
  }
}

resource "opensearch_role" "security_role" {
  role_name           = "monitor_security_role"
  cluster_permissions = ["cluster_monitor"]
}

resource "opensearch_roles_mapping" "security_role" {
  role_name     = "${opensearch_role.security_role.id}"
  backend_roles = ["monitor_role"]
}
	`, resourceName)
}

func testAccOpenDistroUserResourceMinimal(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_user" "test" {
  username = "%s"
  password = "passw0rd"
}
	`, resourceName)
}

func testAccOpenDistroUserMultiple(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_user" "testuser1" {
  username    = "%s-testuser1"
  password    = "testuser1"
  description = "testuser1"
}

resource "opensearch_user" "testuser2" {
  username    = "%s-testuser2"
  password    = "testuser2"
  description = "testuser2"
}

resource "opensearch_user" "testuser3" {
  username    = "%s-testuser3"
  password    = "testuser3"
  description = "testuser3"
}
	`, resourceName, resourceName, resourceName)
}
