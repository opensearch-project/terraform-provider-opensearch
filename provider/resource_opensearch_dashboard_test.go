package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchDashboard(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	randomID := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDashboardsPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testAccCheckOpensearchDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDashboardResource(randomID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpensearchDashboardExists("opensearch_dashboard.test"),
					resource.TestCheckResourceAttr("opensearch_dashboard.test", "objects.#", "1"),
					resource.TestCheckResourceAttr("opensearch_dashboard.test", "objects.0.type", "index-pattern"),
					resource.TestCheckResourceAttr("opensearch_dashboard.test", "objects.0.id", randomID),
				),
			},
			{
				Config: testAccOpensearchDashboardResourceUpdated(randomID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpensearchDashboardExists("opensearch_dashboard.test"),
					resource.TestCheckResourceAttr("opensearch_dashboard.test", "objects.#", "2"),
				),
			},
			{
				// The index-pattern from the previous step is no longer
				// present in source, so it should be deleted on update.
				Config: testAccOpensearchDashboardResourceRemoved(randomID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpensearchDashboardExists("opensearch_dashboard.test"),
					resource.TestCheckResourceAttr("opensearch_dashboard.test", "objects.#", "1"),
					resource.TestCheckResourceAttr("opensearch_dashboard.test", "objects.0.type", "search"),
					testAccCheckOpensearchDashboardObjectGone("index-pattern", randomID),
				),
			},
		},
	})
}

func testAccDashboardsPreCheck(t *testing.T) {
	if v := os.Getenv("OPENSEARCH_DASHBOARDS_URL"); v == "" {
		t.Skip("OPENSEARCH_DASHBOARDS_URL must be set for opensearch_dashboard acceptance tests")
	}
}

func dashboardRefsFromAttributes(attrs map[string]string) []dashboardSavedObjectRef {
	count, _ := strconv.Atoi(attrs["objects.#"])
	refs := make([]dashboardSavedObjectRef, 0, count)
	for i := 0; i < count; i++ {
		refs = append(refs, dashboardSavedObjectRef{
			Type: attrs[fmt.Sprintf("objects.%d.type", i)],
			ID:   attrs[fmt.Sprintf("objects.%d.id", i)],
		})
	}
	return refs
}

func testAccCheckOpensearchDashboardExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no id is set")
		}

		conf := testAccOpendistroProvider.Meta().(*ProviderConf)
		for _, ref := range dashboardRefsFromAttributes(rs.Primary.Attributes) {
			if err := dashboardGetSavedObject(conf, ref); err != nil {
				return fmt.Errorf("saved object (%s/%s) not found: %+v", ref.Type, ref.ID, err)
			}
		}
		return nil
	}
}

func testAccCheckOpensearchDashboardObjectGone(objType, id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)
		err := dashboardGetSavedObject(conf, dashboardSavedObjectRef{Type: objType, ID: id})
		if err == nil {
			return fmt.Errorf("saved object (%s/%s) still exists, expected it to have been removed", objType, id)
		}
		if !isDashboardsNotFound(err) {
			return err
		}
		return nil
	}
}

func testAccCheckOpensearchDashboardDestroy(s *terraform.State) error {
	conf := testAccOpendistroProvider.Meta().(*ProviderConf)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_dashboard" {
			continue
		}
		for _, ref := range dashboardRefsFromAttributes(rs.Primary.Attributes) {
			err := dashboardGetSavedObject(conf, ref)
			if err == nil {
				return fmt.Errorf("saved object (%s/%s) still exists", ref.Type, ref.ID)
			}
			if !isDashboardsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccOpensearchDashboardResource(id string) string {
	return fmt.Sprintf(`
resource "opensearch_dashboard" "test" {
  source = <<-EOF
  {"type":"index-pattern","id":"%[1]s","attributes":{"title":"%[1]s-*","timeFieldName":"@timestamp"},"references":[]}
  EOF
}
	`, id)
}

func testAccOpensearchDashboardResourceUpdated(id string) string {
	return fmt.Sprintf(`
resource "opensearch_dashboard" "test" {
  source = <<-EOF
  {"type":"index-pattern","id":"%[1]s","attributes":{"title":"%[1]s-updated-*","timeFieldName":"@timestamp"},"references":[]}
  {"type":"search","id":"%[1]s-search","attributes":{"title":"%[1]s search","columns":[],"sort":[]},"references":[{"name":"kibanaSavedObjectMeta.searchSourceJSON.index","type":"index-pattern","id":"%[1]s"}]}
  EOF
}
	`, id)
}

func testAccOpensearchDashboardResourceRemoved(id string) string {
	return fmt.Sprintf(`
resource "opensearch_dashboard" "test" {
  source = <<-EOF
  {"type":"search","id":"%[1]s-search","attributes":{"title":"%[1]s search","columns":[],"sort":[]},"references":[]}
  EOF
}
	`, id)
}
