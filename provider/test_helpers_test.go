package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Retrieves a resource ID from Terraform state by name or an error if not found or ID is empty.
func getResourceIDFromState(s *terraform.State, name string) (string, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok {
		return "", fmt.Errorf("resource with name '%s' not found in state", name)
	}
	if rs.Primary.ID == "" {
		return "", fmt.Errorf("resource with name '%s' found in state, but with no ID set", name)
	}
	return rs.Primary.ID, nil
}
