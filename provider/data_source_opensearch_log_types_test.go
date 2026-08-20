package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpensearchLogTypes_all(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchLogTypesConfigAll(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_log_types.all", "log_types.#"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchLogTypes_filterBySource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchLogTypesConfigFilterBySource(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_log_types.custom", "log_types.#"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchLogTypes_filterByName(t *testing.T) {
	resourceName := "test_log_type_data_source"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchLogTypesConfigFilterByName(resourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opensearch_log_types.specific", "log_types.#", "1"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.specific", "log_types.0.name", "test-log-type-ds"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.specific", "log_types.0.description", "Test log type for data source"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.specific", "log_types.0.source", "Custom"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.specific", "log_types.0.category", "Applications"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchLogTypes_filterByCategory(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchLogTypesConfigFilterByCategory(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_log_types.security", "log_types.#"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchLogTypes_multipleFilters(t *testing.T) {
	resourceName := "test_log_type_multi_filter"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchLogTypesConfigMultipleFilters(resourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opensearch_log_types.filtered", "log_types.#", "1"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.filtered", "log_types.0.source", "Custom"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.filtered", "log_types.0.category", "Security"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchLogTypes_filterById(t *testing.T) {
	resourceName := "test_log_type_by_id"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchLogTypesConfigFilterById(resourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opensearch_log_types.by_id", "log_types.#", "1"),
					resource.TestCheckResourceAttrPair("data.opensearch_log_types.by_id", "log_types.0.id", "opensearch_log_type.test", "id"),
					resource.TestCheckResourceAttr("data.opensearch_log_types.by_id", "log_types.0.name", "test-log-type-by-id"),
				),
			},
		},
	})
}

func testAccDataSourceOpensearchLogTypesConfigAll() string {
	return `
data "opensearch_log_types" "all" {}
`
}

func testAccDataSourceOpensearchLogTypesConfigFilterBySource() string {
	return `
data "opensearch_log_types" "custom" {
  source = "Custom"
}
`
}

func testAccDataSourceOpensearchLogTypesConfigFilterByName(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_log_type" "%s" {
  name        = "test-log-type-ds"
  description = "Test log type for data source"
  source      = "Custom"
  category    = "Applications"
}

data "opensearch_log_types" "specific" {
  name       = opensearch_log_type.%s.name
  depends_on = [opensearch_log_type.%s]
}
`, resourceName, resourceName, resourceName)
}

func testAccDataSourceOpensearchLogTypesConfigFilterByCategory() string {
	return `
data "opensearch_log_types" "security" {
  category = "Security"
}
`
}

func testAccDataSourceOpensearchLogTypesConfigMultipleFilters(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_log_type" "%s" {
  name        = "test-log-type-multi"
  description = "Test log type with multiple filters"
  source      = "Custom"
  category    = "Security"
}

data "opensearch_log_types" "filtered" {
  source     = "Custom"
  category   = "Security"
  depends_on = [opensearch_log_type.%s]
}
`, resourceName, resourceName)
}

func testAccDataSourceOpensearchLogTypesConfigFilterById(resourceName string) string {
	return fmt.Sprintf(`
resource "opensearch_log_type" "%s" {
  name        = "test-log-type-by-id"
  description = "Test log type for ID filtering"
  source      = "Custom"
  category    = "Other"
}

data "opensearch_log_types" "by_id" {
  id         = opensearch_log_type.%s.id
  depends_on = [opensearch_log_type.%s]
}
`, resourceName, resourceName, resourceName)
}
