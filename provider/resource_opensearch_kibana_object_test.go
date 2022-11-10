package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchKibanaObject(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	var visualizationConfig string
	var indexPatternConfig string
	meta := provider.Meta()
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		t.Skipf("err: %s", err)
	}
	switch esClient.(type) {
	case *elastic7.Client:
		visualizationConfig = testAccOpensearch7KibanaVisualization
		indexPatternConfig = testAccOpensearch7KibanaIndexPattern
	case *elastic6.Client:
		visualizationConfig = testAccOpensearch6KibanaVisualization
		indexPatternConfig = testAccOpensearch6KibanaIndexPattern
	default:
		visualizationConfig = testAccOpensearchKibanaVisualization
		indexPatternConfig = testAccOpensearchKibanaIndexPattern
	}

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchKibanaObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: visualizationConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchKibanaObjectExists("opensearch_kibana_object.test_visualization", "visualization", "response-time-percentile"),
				),
			},
			{
				Config: indexPatternConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchKibanaObjectExists("opensearch_kibana_object.test_pattern", "index-pattern", "index-pattern:cloudwatch"),
				),
			},
		},
	})
}

func TestAccOpensearchKibanaObject_ProviderFormatInvalid(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchKibanaObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchFormatInvalid,
				ExpectError: regexp.MustCompile("must be an array of objects"),
			},
		},
	})
}

func TestAccOpensearchKibanaObject_Rejected(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	meta := provider.Meta()
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		t.Skipf("err: %s", err)
	}
	var allowed bool

	switch esClient.(type) {
	case *elastic6.Client:
		allowed = true
	default:
		allowed = false
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("Only >= ES 6 has index type restrictions")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchKibanaObjectDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchKibanaIndexPattern,
				ExpectError: regexp.MustCompile("Error 400"),
			},
		},
	})
}

func testCheckOpensearchKibanaObjectExists(name string, objectType string, id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No kibana object ID is set")
		}

		meta := testAccProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch client := esClient.(type) {
		case *elastic7.Client:
			_, err = client.Get().Index(".kibana").Id(id).Do(context.TODO())
		case *elastic6.Client:
			_, err = client.Get().Index(".kibana").Type(deprecatedDocType).Id(id).Do(context.TODO())
		default:
			return errors.New("opensearch version not supported")
		}

		if err != nil {
			log.Printf("[INFO] testCheckOpensearchKibanaObjectExists: %+v", err)
			return err
		}

		return nil
	}
}

func testCheckOpensearchKibanaObjectDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_kibana_object" {
			continue
		}

		meta := testAccProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch client := esClient.(type) {
		case *elastic7.Client:
			_, err = client.Get().Index(".kibana").Id("response-time-percentile").Do(context.TODO())
		case *elastic6.Client:
			_, err = client.Get().Index(".kibana").Type("visualization").Id("response-time-percentile").Do(context.TODO())
		default:
			return errors.New("opensearch version not supported")
		}

		if err != nil {
			if elastic7.IsNotFound(err) || elastic6.IsNotFound(err) {
				return nil // should be not found error
			}

			// Fail on any other error
			return fmt.Errorf("Unexpected error %s", err)
		}

		return fmt.Errorf("Kibana object %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchKibanaVisualization = `
resource "opensearch_kibana_object" "test_visualization" {
  body = <<EOF
[
  {
    "_id": "response-time-percentile",
    "_type": "visualization",
    "_source": {
      "title": "Total response time percentiles",
      "visState": "{\"title\":\"Total response time percentiles\",\"type\":\"line\",\"params\":{\"addTooltip\":true,\"addLegend\":true,\"legendPosition\":\"right\",\"showCircles\":true,\"interpolate\":\"linear\",\"scale\":\"linear\",\"drawLinesBetweenPoints\":true,\"radiusRatio\":9,\"times\":[],\"addTimeMarker\":false,\"defaultYExtents\":false,\"setYExtents\":false},\"aggs\":[{\"id\":\"1\",\"enabled\":true,\"type\":\"percentiles\",\"schema\":\"metric\",\"params\":{\"field\":\"app.total_time\",\"percents\":[50,90,95]}},{\"id\":\"2\",\"enabled\":true,\"type\":\"date_histogram\",\"schema\":\"segment\",\"params\":{\"field\":\"@timestamp\",\"interval\":\"auto\",\"customInterval\":\"2h\",\"min_doc_count\":1,\"extended_bounds\":{}}},{\"id\":\"3\",\"enabled\":true,\"type\":\"terms\",\"schema\":\"group\",\"params\":{\"field\":\"system.syslog.program\",\"size\":5,\"order\":\"desc\",\"orderBy\":\"_term\"}}],\"listeners\":{}}",
      "uiStateJSON": "{}",
      "description": "",
      "version": 1,
      "kibanaSavedObjectMeta": {
        "searchSourceJSON": "{\"index\":\"filebeat-*\",\"query\":{\"query_string\":{\"query\":\"*\",\"analyze_wildcard\":true}},\"filter\":[]}"
      }
    }
  }
]
EOF
}
`

var testAccOpensearch6KibanaVisualization = `
resource "opensearch_kibana_object" "test_visualization" {
  body = <<EOF
[
  {
    "_id": "response-time-percentile",
    "_type": "doc",
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

var testAccOpensearch7KibanaVisualization = `
resource "opensearch_kibana_object" "test_visualization" {
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

var testAccOpensearchKibanaIndexPattern = `
resource "opensearch_kibana_object" "test_pattern" {
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

var testAccOpensearch6KibanaIndexPattern = `
resource "opensearch_kibana_object" "test_pattern" {
  body = <<EOF
[
  {
		"_id": "index-pattern:cloudwatch",
		"_type": "doc",
		"_source": {
			"type": "index-pattern",
			"index-pattern": {
				"title": "cloudwatch-*",
				"timeFieldName": "timestamp"
			}
		}
	}
]
EOF
}
`

var testAccOpensearch7KibanaIndexPattern = `
resource "opensearch_kibana_object" "test_pattern" {
  body = <<EOF
[
  {
		"_id": "index-pattern:cloudwatch",
		"_type": "_doc",
		"_source": {
			"type": "index-pattern",
			"index-pattern": {
				"title": "cloudwatch-*",
				"timeFieldName": "timestamp"
			}
		}
	}
]
EOF
}
`

var testAccOpensearchFormatInvalid = `
resource "opensearch_kibana_object" "test_invalid" {
  body = <<EOF
{
  "test": "yes"
}
EOF
}
`
