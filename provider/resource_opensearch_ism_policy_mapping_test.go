package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroISMPolicyMapping(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool = true

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// TODO add check for OpenDistro <= 1.13
			if !allowed {
				t.Skip("OpenDistroISMPolicies only supported on ES 2.0.0")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyMappingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchOpenDistroISMPolicyMapping,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchISMPolicyMappingExists("opensearch_ism_policy_mapping.test_mapping"),
					resource.TestCheckResourceAttr(
						"opensearch_ism_policy_mapping.test_mapping",
						"policy_id",
						"test_policy",
					),
				),
			},
			{
				Config: testAccOpensearchOpenDistroISMPolicyMappingUpdate,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchISMPolicyMappingExists("opensearch_ism_policy_mapping.test_mapping"),
					resource.TestCheckResourceAttr(
						"opensearch_ism_policy_mapping.test_mapping",
						"policy_id",
						"test_policy",
					),
				),
			},
		},
	})
}

func testCheckOpensearchISMPolicyMappingExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy ID is set")
		}

		_, err := indicesMappedToPolicy(rs.Primary.ID)

		if err != nil {
			return err
		}

		return nil
	}
}

func indicesMappedToPolicy(policy string) ([]string, error) {
	meta := testAccOpendistroProvider.Meta()

	var err error
	var indices map[string]interface{}
	mappedIndices := []string{}
	indices, err = resourceOpensearchGetOpendistroPolicyMapping(policy, meta.(*ProviderConf))

	if err != nil {
		return mappedIndices, err
	}

	for indexName, parameters := range indices {
		p, ok := parameters.(map[string]interface{})
		if ok && p["index.opendistro.index_state_management.policy_id"] == policy {
			mappedIndices = append(mappedIndices, indexName)
		} else if ok && p["index.plugins.index_state_management.policy_id"] == policy {
			mappedIndices = append(mappedIndices, indexName)
		}
	}
	return mappedIndices, nil
}

func testCheckOpensearchISMPolicyMappingDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_ism_policy_mapping" {
			continue
		}

		mappedIndices, err := indicesMappedToPolicy(rs.Primary.ID)

		// if the underlying index is deleted, it triggers a cascading delete for
		// the mapping and the mapping explain endpoint returns a 400, so we know
		// it's been cleaned up
		var e *elastic7.Error
		if err != nil && errors.As(err, &e) && e.Status == 400 {
			return nil
		}

		if err != nil {
			return err
		}

		if len(mappedIndices) == 0 {
			return nil
		}

		return fmt.Errorf("OpenDistroISMPolicyMapping %q still exists: %+v", rs.Primary.ID, mappedIndices)
	}

	return nil
}

var testAccOpensearchOpenDistroISMPolicyMapping = `
resource "opensearch_ism_policy" "test_policy" {
  policy_id = "test_policy"
  body      = <<EOF
 {
	"policy": {
	  "description": "ingesting logs into ${opensearch_index.test.name}",
	  "default_state": "ingest",
	  "states": [
			{
			  "name": "ingest",
			  "actions": [],
			  "transitions": [{
				  "state_name": "search"
				}]
			},
			{
			  "name": "search",
			  "actions": [],
			  "transitions": []
			}
		]
	}
 }
 EOF
}

resource "opensearch_index" "test" {
  name               = "ingest-0001"
  number_of_shards   = 1
  number_of_replicas = 1
}

resource "opensearch_ism_policy_mapping" "test_mapping" {
  policy_id = "${opensearch_ism_policy.test_policy.id}"
  indexes   = "ingest-*"
}
`

var testAccOpensearchOpenDistroISMPolicyMappingUpdate = `
resource "opensearch_ism_policy" "test_policy" {
  policy_id = "test_policy"
  body      = <<EOF
 {
	"policy": {
	  "description": "ingesting logs into ${opensearch_index.test.name}",
	  "default_state": "ingest",
	  "states": [
			{
			  "name": "ingest",
			  "actions": [],
			  "transitions": [{
				  "state_name": "search"
				}]
			},
			{
			  "name": "search",
			  "actions": [],
			  "transitions": []
			}
		]
	}
 }
 EOF
}

resource "opensearch_index" "test" {
  name               = "ingest-0001"
  number_of_shards   = 1
  number_of_replicas = 1
}

resource "opensearch_ism_policy_mapping" "test_mapping" {
  policy_id = "${opensearch_ism_policy.test_policy.id}"
  indexes   = "ingest-*"
  state     = "search"
  include = [{
    state = "ingest"
  }]
}
`
