package provider

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestParseDashboardObjectImportID(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		objectID   string
		tenantName string
		indexName  string
		expectErr  bool
	}{
		{
			name:      "default",
			input:     "response-time-percentile",
			objectID:  "response-time-percentile",
			indexName: "",
		},
		{
			name:       "tenant name",
			input:      "response-time-percentile,tenant_name=tenant_test",
			objectID:   "response-time-percentile",
			tenantName: "tenant_test",
		},
		{
			name:      "custom index",
			input:     "response-time-percentile,index=.kibana_custom",
			objectID:  "response-time-percentile",
			indexName: ".kibana_custom",
		},
		{
			name:      "empty object id",
			input:     ",tenant_name=tenant_test",
			expectErr: true,
		},
		{
			name:      "unknown key",
			input:     "response-time-percentile,foo=bar",
			expectErr: true,
		},
		{
			name:      "conflicting keys",
			input:     "response-time-percentile,tenant_name=tenant_test,index=.kibana_custom",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objectID, tenantName, indexName, err := parseDashboardObjectImportID(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if objectID != tc.objectID {
				t.Fatalf("expected object_id %q, got %q", tc.objectID, objectID)
			}
			if tenantName != tc.tenantName {
				t.Fatalf("expected tenant_name %q, got %q", tc.tenantName, tenantName)
			}
			if indexName != tc.indexName {
				t.Fatalf("expected index %q, got %q", tc.indexName, indexName)
			}
		})
	}
}

func TestAccOpensearchDashboardObject(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	visualizationConfig := testAccOpensearch7DashboardVisualization
	indexPatternConfig := testAccOpensearch7DashboardIndexPattern

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchDashboardObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: visualizationConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDashboardObjectExists("opensearch_dashboard_object.test_visualization", "response-time-percentile", ""),
				),
			},
			{
				ResourceName:      "opensearch_dashboard_object.test_visualization",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "response-time-percentile",
			},
			{
				Config: indexPatternConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDashboardObjectExists("opensearch_dashboard_object.test_pattern", "index-pattern:cloudwatch", ""),
				),
			},
		},
	})
}

func TestAccOpensearchDashboardObjectWithTenant(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	visualizationConfig := testAccOpensearch7DashboardVisualizationWithTenant
	indexPatternConfig := testAccOpensearch7DashboardIndexPatternWithTenant

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchDashboardObjectDestroyWithTenant,
		Steps: []resource.TestStep{
			{
				Config: visualizationConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opensearch_dashboard_tenant.tenant_test", "tenant_name", "tenant_test"),
					testCheckOpensearchDashboardObjectExists("opensearch_dashboard_object.test_visualization", "response-time-percentile", "tenant_test"),
				),
			},
			{
				ResourceName:      "opensearch_dashboard_object.test_visualization",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "response-time-percentile,tenant_name=tenant_test",
			},
			{
				Config: indexPatternConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opensearch_dashboard_tenant.tenant_test", "tenant_name", "tenant_test"),
					testCheckOpensearchDashboardObjectExists("opensearch_dashboard_object.test_pattern", "index-pattern:cloudwatch", "tenant_test"),
				),
			},
		},
	})
}

func TestAccOpensearchDashboardObject_ProviderFormatInvalid(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchDashboardObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchFormatInvalid,
				ExpectError: regexp.MustCompile("must be an array of objects"),
			},
		},
	})
}

func TestAccOpensearchDashboardObject_Rejected(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed = false

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("Only >= OS 2.0.0 has index type restrictions")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchDashboardObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchDashboardIndexPattern,
				ExpectError: regexp.MustCompile("Error 400"),
			},
		},
	})
}

func testCheckOpensearchDashboardObjectExists(name string, id string, tenantName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Dashboard object ID is set")
		}

		meta := testAccProvider.Meta()

		var err error
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = osClient.Get().
			Index(".kibana").
			Id(id).
			Header(SECURITY_TENANT_HEADER, tenantName).
			Do(context.TODO())

		if err != nil {
			log.Printf("[INFO] testCheckOpensearchDashboardObjectExists: %+v", err)
			return err
		}

		return nil
	}
}

func testCheckOpensearchDashboardObjectDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_dashboard_object" {
			continue
		}

		meta := testAccProvider.Meta()
		tenantName := rs.Primary.Attributes["tenant_name"]

		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = osClient.Get().
			Index(".kibana").
			Id("response-time-percentile").
			Header(SECURITY_TENANT_HEADER, tenantName).
			Do(context.TODO())

		if err != nil {
			if elastic7.IsNotFound(err) {
				return nil // should be not found error
			}

			if tenantName != "global_tenant" && (elastic7.IsForbidden(err)) {
				// when tenant has been destroyed this is the expected error
				return nil
			}

			// Fail on any other error
			return fmt.Errorf("Unexpected error %s", err)
		}

		return fmt.Errorf("Dashboard object %q still exists", rs.Primary.ID)
	}

	return nil
}

func testCheckOpensearchDashboardObjectDestroyWithTenant(s *terraform.State) error {
	if err := testCheckOpensearchDashboardObjectDestroy(s); err != nil {
		return err
	}
	if err := testAccCheckOpensearchDashboardTenantDestroy(s); err != nil {
		return err
	}
	return nil
}

var testAccOpensearch7DashboardVisualization = `
resource "opensearch_dashboard_object" "test_visualization" {
  body = <<EOF
[
  {
    "_id": "response-time-percentile",
    "_source": {
      "visualization": {
	      "title": "Total response time percentiles",
	      "visState": "{\"title\":\"Total response time percentiles\",\"type\":\"line\",\"params\":{\"addTooltip\":true,\"addLegend\":true,\"legendPosition\":\"right\",\"showCircles\":true,\"interpolate\":\"linear\",\"scale\":\"linear\",\"drawLinesBetweenPoints\":true,\"radiusRatio\":9,\"times\":[],\"addTimeMarker\":false,\"defaultYExtents\":false,\"setYExtents\":false},\"aggs\":[{\"id\":\"1\",\"enabled\":true,\"type\":\"percentiles\",\"schema\":\"metric\",\"params\":{\"field\":\"app.total_time\",\"percents\":[50,90,95]}},{\"id\":\"2\",\"enabled\":true,\"type\":\"date_histogram\",\"schema\":\"segment\",\"params\":{\"field\":\"@timestamp\",\"interval\":\"auto\",\"customInterval\":\"2h\",\"min_doc_count\":1,\"extended_bounds\":{}}},{\"id\":\"3\",\"enabled\":true,\"type\":\"terms\",\"schema\":\"group\",\"params\":{\"field\":\"system.syslog.program\",\"size\":5,\"order\":\"desc\",\"orderBy\":\"_term\"}}],\"listeners\":{}}",
	      "uiStateJSON": "{}",
	      "description": "",
	      "version": 1,
	      "kibanaSavedObjectMeta": {
	        "searchSourceJSON": "{\"index\":\"filebeat-*\",\"query\":{\"query_string\":{\"query\":\"*\",\"analyze_wildcard\":true}},\"filter\":[]}"
	      }
	    },
      "type": "visualization"
    }
  }
]
EOF
}
`

var testAccOpensearch7DashboardVisualizationWithTenant = `
resource "opensearch_dashboard_tenant" "tenant_test" {
  tenant_name = "tenant_test"
  description = "tenant_test"
}

resource "opensearch_dashboard_object" "test_visualization" {
  depends_on = [
    opensearch_dashboard_tenant.tenant_test
  ]
  tenant_name = "tenant_test"
  body        = <<EOF
[
  {
    "_id": "response-time-percentile",
    "_source": {
      "visualization": {
	      "title": "Total response time percentiles",
	      "visState": "{\"title\":\"Total response time percentiles\",\"type\":\"line\",\"params\":{\"addTooltip\":true,\"addLegend\":true,\"legendPosition\":\"right\",\"showCircles\":true,\"interpolate\":\"linear\",\"scale\":\"linear\",\"drawLinesBetweenPoints\":true,\"radiusRatio\":9,\"times\":[],\"addTimeMarker\":false,\"defaultYExtents\":false,\"setYExtents\":false},\"aggs\":[{\"id\":\"1\",\"enabled\":true,\"type\":\"percentiles\",\"schema\":\"metric\",\"params\":{\"field\":\"app.total_time\",\"percents\":[50,90,95]}},{\"id\":\"2\",\"enabled\":true,\"type\":\"date_histogram\",\"schema\":\"segment\",\"params\":{\"field\":\"@timestamp\",\"interval\":\"auto\",\"customInterval\":\"2h\",\"min_doc_count\":1,\"extended_bounds\":{}}},{\"id\":\"3\",\"enabled\":true,\"type\":\"terms\",\"schema\":\"group\",\"params\":{\"field\":\"system.syslog.program\",\"size\":5,\"order\":\"desc\",\"orderBy\":\"_term\"}}],\"listeners\":{}}",
	      "uiStateJSON": "{}",
	      "description": "",
	      "version": 1,
	      "kibanaSavedObjectMeta": {
	        "searchSourceJSON": "{\"index\":\"filebeat-*\",\"query\":{\"query_string\":{\"query\":\"*\",\"analyze_wildcard\":true}},\"filter\":[]}"
	      }
	    },
      "type": "visualization"
    }
  }
]
EOF
}
`

var testAccOpensearchDashboardIndexPattern = `
resource "opensearch_dashboard_object" "test_pattern" {
  body = <<EOF
[
  {
		"_id": "index-pattern:cloudwatch",
		"_type": "index-pattern",
		"_source": {
			"title": "cloudwatch-*",
			"timeFieldName": "timestamp"
		}
	}
]
EOF
}
`

var testAccOpensearch7DashboardIndexPattern = `
resource "opensearch_dashboard_object" "test_pattern" {
  index = ".kibana"
  body  = <<EOF
[
  {
		"_id": "index-pattern:cloudwatch",
		"_source": {
			"type": "index-pattern",
			"index-pattern": {
				"title": "cloudwatch-*",
				"timeFieldName": "@timestamp"
			}
		}
	}
]
EOF
}
`

var testAccOpensearch7DashboardIndexPatternWithTenant = `
resource "opensearch_dashboard_tenant" "tenant_test" {
  tenant_name = "tenant_test"
  description = "tenant_test"
}

resource "opensearch_dashboard_object" "test_pattern" {
  depends_on = [
    opensearch_dashboard_tenant.tenant_test
  ]
  tenant_name = "tenant_test"
  body        = <<EOF
[
  {
		"_id": "index-pattern:cloudwatch",
		"_source": {
			"type": "index-pattern",
			"index-pattern": {
				"title": "cloudwatch-*",
				"timeFieldName": "@timestamp"
			}
		}
	}
]
EOF
}
`

var testAccOpensearchFormatInvalid = `
resource "opensearch_dashboard_object" "test_invalid" {
  body = <<EOF
{
  "test": "yes"
}
EOF
}
`

func indexPatternBody(fields string) string {
	return indexPatternBodyAt(fields, "2026-01-01T00:00:00.000Z")
}

func indexPatternBodyAt(fields, updatedAt string) string {
	return `[{"_id":"index-pattern:logs","_source":{"type":"index-pattern","index-pattern":{"title":"logs-*","timeFieldName":"@timestamp","fields":"` + fields + `"},"references":[],"updated_at":"` + updatedAt + `"}}]`
}

func TestDashboardObjectBookkeepingDiffSuppress(t *testing.T) {
	// Popularity counters live inside the JSON-encoded `fields` attribute, so the escaping
	// here is what an export actually contains.
	noPopularity := `[{\"count\":0,\"name\":\"@timestamp\",\"type\":\"date\"},{\"count\":0,\"name\":\"message\",\"type\":\"string\"}]`
	somePopularity := `[{\"count\":7,\"name\":\"@timestamp\",\"type\":\"date\"},{\"count\":3,\"name\":\"message\",\"type\":\"string\"}]`
	renamedField := `[{\"count\":0,\"name\":\"@timestamp\",\"type\":\"date\"},{\"count\":0,\"name\":\"msg\",\"type\":\"string\"}]`

	visualization := `[{"_id":"visualization:errors","_source":{"type":"visualization","visualization":{"title":"Errors","visState":"{\"aggs\":[{\"type\":\"count\"}]}"},"references":[]}}]`
	visualizationAvg := `[{"_id":"visualization:errors","_source":{"type":"visualization","visualization":{"title":"Errors","visState":"{\"aggs\":[{\"type\":\"avg\"}]}"},"references":[]}}]`

	testCases := []struct {
		name     string
		old      string
		new      string
		suppress bool
	}{
		{
			name:     "popularity climbed since export",
			old:      indexPatternBody(somePopularity),
			new:      indexPatternBody(noPopularity),
			suppress: true,
		},
		{
			name:     "identical bodies",
			old:      indexPatternBody(somePopularity),
			new:      indexPatternBody(somePopularity),
			suppress: true,
		},
		{
			name:     "field renamed as well as popularity",
			old:      indexPatternBody(somePopularity),
			new:      indexPatternBody(renamedField),
			suppress: false,
		},
		{
			name:     "title changed",
			old:      indexPatternBody(somePopularity),
			new:      strings.Replace(indexPatternBody(noPopularity), "logs-*", "logs-2024-*", 1),
			suppress: false,
		},
		{
			name: "count in a non index-pattern object is left alone",
			// `count` here is an aggregation type, not a popularity counter.
			old:      visualization,
			new:      visualizationAvg,
			suppress: false,
		},
		{
			// What Dashboards actually does: bumping a counter restamps updated_at too.
			name:     "popularity climbed and updated_at was restamped",
			old:      indexPatternBodyAt(somePopularity, "2026-08-17T21:04:11.884Z"),
			new:      indexPatternBodyAt(noPopularity, "2026-01-01T00:00:00.000Z"),
			suppress: true,
		},
		{
			name:     "updated_at alone",
			old:      indexPatternBodyAt(somePopularity, "2026-08-17T21:04:11.884Z"),
			new:      indexPatternBodyAt(somePopularity, "2026-01-01T00:00:00.000Z"),
			suppress: true,
		},
		{
			name:     "updated_at restamped alongside a real edit",
			old:      indexPatternBodyAt(somePopularity, "2026-08-17T21:04:11.884Z"),
			new:      strings.Replace(indexPatternBodyAt(noPopularity, "2026-01-01T00:00:00.000Z"), "logs-*", "logs-2024-*", 1),
			suppress: false,
		},
		{
			// migrationVersion is real content, so it must still diff.
			name:     "migrationVersion changed",
			old:      strings.Replace(indexPatternBody(somePopularity), `"references":[]`, `"references":[],"migrationVersion":{"index-pattern":"7.6.0"}`, 1),
			new:      strings.Replace(indexPatternBody(noPopularity), `"references":[]`, `"references":[],"migrationVersion":{"index-pattern":"7.11.0"}`, 1),
			suppress: false,
		},
		{
			name:     "unparseable body is not suppressed",
			old:      "not json",
			new:      indexPatternBody(noPopularity),
			suppress: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dashboardObjectBookkeepingDiffSuppress("body", tc.old, tc.new, nil); got != tc.suppress {
				t.Fatalf("expected suppress %v, got %v", tc.suppress, got)
			}
		})
	}
}

func TestStripDashboardBookkeeping(t *testing.T) {
	stripped, err := stripDashboardBookkeeping(indexPatternBody(`[{\"count\":7,\"name\":\"@timestamp\"}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, unwanted := range []string{"count", "updated_at"} {
		if strings.Contains(stripped, unwanted) {
			t.Fatalf("expected %q to be removed, got %s", unwanted, stripped)
		}
	}
	// Everything else about the object has to survive, or unrelated drift would be hidden too.
	for _, want := range []string{"index-pattern:logs", "logs-*", "@timestamp", "timeFieldName"} {
		if !strings.Contains(stripped, want) {
			t.Fatalf("expected %q to survive stripping, got %s", want, stripped)
		}
	}
}
