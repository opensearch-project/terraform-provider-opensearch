package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Map of resource IDS to verify resources are not recreated on updates.
// Note: This global map is shared across tests.
var savedMLModelGroupIDs = make(map[string]string)

func testSaveOpensearchMLModelGroupID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		savedMLModelGroupIDs[name] = resourceID
		return nil
	}
}

func testCheckOpensearchMLModelGroupIDEqualsSavedID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		currentID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}

		savedID, ok := savedMLModelGroupIDs[name]
		if !ok {
			return fmt.Errorf("resource with name '%s' not found in savedMLModelGroupIDs", name)
		}

		if savedID != currentID {
			return fmt.Errorf("ID of ML Model Group with name %s does not match original resource. ID of original resource: %s. ID of resource after update: %s", name, savedID, currentID)
		}

		return nil
	}
}

func TestAccOpensearchMLModelGroup_Minimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelGroupConfig_Minimal(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.minimal", "name", "minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.minimal", "access_mode", "private"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.minimal", "backend_roles.#", "0"),
				),
			},
			{
				ResourceName:      "opensearch_ml_model_group.minimal",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLModelGroup_WithOptionalAttributes(t *testing.T) {
	defer func() {
		delete(savedMLModelGroupIDs, "opensearch_ml_model_group.with_optional")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelGroupConfig_WithOptionalAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.with_optional"),
					testSaveOpensearchMLModelGroupID("opensearch_ml_model_group.with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_optional", "name", "with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_optional", "description", "ML Model Group with optional attributes"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_optional", "access_mode", "private"),
				),
			},
			{
				Config: testAccOpensearchMLModelGroupConfig_WithOptionalAttributes_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.with_optional"),
					testCheckOpensearchMLModelGroupIDEqualsSavedID("opensearch_ml_model_group.with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_optional", "name", "with_optional_updated"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_optional", "description", "Updated description for ML Model Group"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_optional", "access_mode", "public"),
				),
			},
			{
				ResourceName:      "opensearch_ml_model_group.with_optional",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLModelGroup_WithAccessModes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelGroupConfig_Public(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.public"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.public", "name", "public"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.public", "access_mode", "public"),
				),
			},
			{
				Config: testAccOpensearchMLModelGroupConfig_Private(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.private"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.private", "name", "private"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.private", "access_mode", "private"),
				),
			},
			{
				ResourceName:      "opensearch_ml_model_group.private",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLModelGroup_WithBackendRoles(t *testing.T) {
	defer func() {
		delete(savedMLModelGroupIDs, "opensearch_ml_model_group.with_backend_roles")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelGroupConfig_Restricted_WithBackendRoles(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.with_backend_roles"),
					testSaveOpensearchMLModelGroupID("opensearch_ml_model_group.with_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "name", "with_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "access_mode", "restricted"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "backend_roles.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "backend_roles.0", "ml_full_access"),
				),
			},
			{
				Config: testAccOpensearchMLModelGroupConfig_Restricted_WithBackendRoles_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.with_backend_roles"),
					testCheckOpensearchMLModelGroupIDEqualsSavedID("opensearch_ml_model_group.with_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "name", "with_backend_roles_updated"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "access_mode", "restricted"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "backend_roles.#", "2"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "backend_roles.0", "ml_full_access"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_backend_roles", "backend_roles.1", "ml_readonly_access"),
				),
			},
			{
				ResourceName:      "opensearch_ml_model_group.with_backend_roles",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLModelGroup_WithAddAllBackendRoles(t *testing.T) {
	t.Skip("Skipping test: OpenSearch API restriction - Admin users cannot add all backend roles to a model group")

	defer func() {
		delete(savedMLModelGroupIDs, "opensearch_ml_model_group.with_add_all_backend_roles")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelGroupConfig_Restricted_WithAddAllBackendRoles(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.with_add_all_backend_roles"),
					testSaveOpensearchMLModelGroupID("opensearch_ml_model_group.with_add_all_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "name", "with_add_all_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "access_mode", "restricted"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "backend_roles.#", "0"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "add_all_backend_roles", "true"),
				),
			},
			{
				Config: testAccOpensearchMLModelGroupConfig_Restricted_WithAddAllBackendRoles_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelGroupExists("opensearch_ml_model_group.with_add_all_backend_roles"),
					testCheckOpensearchMLModelGroupIDEqualsSavedID("opensearch_ml_model_group.with_add_all_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "name", "with_add_all_backend_roles_updated"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "access_mode", "restricted"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "backend_roles.#", "0"),
					resource.TestCheckResourceAttr("opensearch_ml_model_group.with_add_all_backend_roles", "add_all_backend_roles", "false"),
				),
			},
			{
				ResourceName:      "opensearch_ml_model_group.with_add_all_backend_roles",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOpensearchMLModelGroup_ConflictingAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchMLModelGroupConfig_ConflictingBackendRoles(),
				ExpectError: regexp.MustCompile("conflicts with"),
			},
		},
	})
}

func testCheckOpensearchMLModelGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMLModelGroupFromAPI(context.Background(), conf, resourceID)
		return apiError
	}
}

func testCheckOpensearchMLModelGroupDestroy(s *terraform.State) error {
	resourceType := "opensearch_ml_model_group"
	for _, rs := range s.RootModule().Resources {
		if rs.Type != resourceType {
			continue
		}

		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMLModelGroupFromAPI(context.Background(), conf, rs.Primary.ID)

		if apiError == nil {
			return fmt.Errorf("resource of type %s with ID '%s' still exists", resourceType, rs.Primary.ID)
		}
		if !strings.Contains(apiError.Error(), "404") {
			return fmt.Errorf("unexpected error verifying resource of type %s with ID '%s' was destroyed: %v", resourceType, rs.Primary.ID, apiError)
		}
	}

	return nil
}

func testAccOpensearchMLModelGroupConfig_Minimal() string {
	return `
resource "opensearch_ml_model_group" "minimal" {
  name = "minimal"
}
`
}

func testAccOpensearchMLModelGroupConfig_WithOptionalAttributes() string {
	return `
resource "opensearch_ml_model_group" "with_optional" {
  name        = "with_optional"
  description = "ML Model Group with optional attributes"
  access_mode = "private"
}
`
}

func testAccOpensearchMLModelGroupConfig_WithOptionalAttributes_Updated() string {
	return `
resource "opensearch_ml_model_group" "with_optional" {
  name        = "with_optional_updated"
  description = "Updated description for ML Model Group"
  access_mode = "public"
}
`
}

func testAccOpensearchMLModelGroupConfig_Public() string {
	return `
resource "opensearch_ml_model_group" "public" {
  name        = "public"
  access_mode = "public"
}
`
}

func testAccOpensearchMLModelGroupConfig_Private() string {
	return `
resource "opensearch_ml_model_group" "private" {
  name        = "private"
  access_mode = "private"
}
`
}

func testAccOpensearchMLModelGroupConfig_Restricted_WithBackendRoles() string {
	return `
resource "opensearch_ml_model_group" "with_backend_roles" {
  name          = "with_backend_roles"
  access_mode   = "restricted"
  backend_roles = ["ml_full_access"]
}
`
}

func testAccOpensearchMLModelGroupConfig_Restricted_WithBackendRoles_Updated() string {
	return `
resource "opensearch_ml_model_group" "with_backend_roles" {
  name          = "with_backend_roles_updated"
  access_mode   = "restricted"
  backend_roles = ["ml_full_access", "ml_readonly_access"]
}
`
}

func testAccOpensearchMLModelGroupConfig_Restricted_WithAddAllBackendRoles() string {
	return `
resource "opensearch_ml_model_group" "restricted_with_add_all_backend_roles" {
  name                  = "restricted_with_add_all_backend_roles"
  access_mode           = "restricted"
  add_all_backend_roles = true
}
`
}

func testAccOpensearchMLModelGroupConfig_Restricted_WithAddAllBackendRoles_Updated() string {
	return `
resource "opensearch_ml_model_group" "restricted_with_add_all_backend_roles" {
  name                  = "restricted_with_add_all_backend_roles_updated"
  access_mode           = "restricted"
  add_all_backend_roles = false
}
`
}

func testAccOpensearchMLModelGroupConfig_ConflictingBackendRoles() string {
	return `
resource "opensearch_ml_model_group" "conflicting" {
  name                  = "conflicting"
  access_mode           = "restricted"
  backend_roles         = ["ml_full_access"]
  add_all_backend_roles = true
}
`
}
