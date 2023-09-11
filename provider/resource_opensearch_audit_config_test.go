package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenSearchSecurityAuditConfig(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccOpenSearchSecurityAuditConfigResource(true),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchSecurityAuditConfigExists("opensearch_audit_config.test"),
					testCheckOpensearchSecurityAuditConfigConnects("opensearch_audit_config.test"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "enabled", "true"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "audit.#", "1"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "audit.0.enable_rest", "true"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "audit.0.disabled_rest_categories.#", "2"),
					resource.TestCheckTypeSetElemAttr("opensearch_audit_config.test", "audit.0.disabled_rest_categories.*", "AUTHENTICATED"),
					resource.TestCheckTypeSetElemAttr("opensearch_audit_config.test", "audit.0.disabled_rest_categories.*", "GRANTED_PRIVILEGES"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "compliance.#", "1"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "compliance.0.enabled", "true"),
				),
			},
			{
				Config: testAccOpenSearchSecurityAuditConfigResourceUpdated(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "enabled", "false"),
					testCheckOpensearchRoleExists("opensearch_audit_config.test"),
					resource.TestCheckResourceAttr("opensearch_audit_config.test", "audit.0.disabled_rest_categories.#", "1"),
				),
			},
		},
	})
}

func testAccOpenSearchSecurityAuditConfigResource(enabled bool) string {
	return fmt.Sprintf(`
resource "opensearch_audit_config" "test" {
  enabled = %t
  audit {
    enable_rest                   = true
    disabled_rest_categories      = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]
    enable_transport              = true
    disabled_transport_categories = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]
    resolve_bulk_requests         = true
    log_request_body              = true
    resolve_indices               = true
    exclude_sensitive_headers     = true
    ignore_users                  = ["dashboardserver"]
    ignore_requests               = ["SearchRequest", "indices:data/read/*", "/_cluster/health"]
  }
  compliance {
    enabled            = true
    internal_config    = true
    external_config    = false
    read_metadata_only = true
    read_ignore_users  = ["read-ignore-1"]
    read_watched_field {
      index  = "read-index-1"
      fields = ["field-1", "field-2"]
    }
    read_watched_field {
      index  = "read-index-2"
      fields = ["field-3"]
    }
    write_metadata_only   = true
    write_log_diffs       = false
    write_watched_indices = ["write-index-1", "write-index-2", "log-*", "*"]
    write_ignore_users    = ["write-ignore-1"]
  }
}`, enabled)
}

func testAccOpenSearchSecurityAuditConfigResourceUpdated(enabled bool) string {
	return fmt.Sprintf(`
resource "opensearch_audit_config" "test" {
  enabled = %t
  audit {
    enable_rest                   = true
    disabled_rest_categories      = ["GRANTED_PRIVILEGES"]
    enable_transport              = true
    disabled_transport_categories = ["GRANTED_PRIVILEGES", "AUTHENTICATED"]
    resolve_bulk_requests         = true
    log_request_body              = true
    resolve_indices               = true
    exclude_sensitive_headers     = true
    ignore_users                  = ["dashboardserver"]
    ignore_requests               = ["SearchRequest", "indices:data/read/*", "/_cluster/health"]
  }
  compliance {
    enabled            = true
    internal_config    = true
    external_config    = false
    read_metadata_only = true
    read_ignore_users  = ["read-ignore-1"]
    read_watched_field {
      index  = "read-index-1"
      fields = ["field-1", "field-2"]
    }
    read_watched_field {
      index  = "read-index-2"
      fields = ["field-3"]
    }
    write_metadata_only   = true
    write_log_diffs       = false
    write_watched_indices = ["write-index-1", "write-index-2", "log-*", "*"]
    write_ignore_users    = ["write-ignore-1"]
  }
}`, enabled)
}

func testCheckOpensearchSecurityAuditConfigExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opensearch_audit_config" {
				continue
			}

			meta := testAccOpendistroProvider.Meta()

			var err error
			if err != nil {
				return err
			}
			_, err = resourceOpensearchGetAuditConfig(meta.(*ProviderConf))

			if err != nil {
				return err
			}

			return nil
		}

		return nil
	}
}

func testCheckOpensearchSecurityAuditConfigConnects(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "opensearch_audit_config" {
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
