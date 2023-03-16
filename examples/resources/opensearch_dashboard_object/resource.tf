
resource "opensearch_dashboard_object" "test_visualization_v6" {
  body = <<EOF
[
  {
    "_id": "visualization:response-time-percentile",
    "_type": "doc",
    "_source": {
      "type": "visualization",
      "visualization": {
        "title": "Total response time percentiles",
        "visState": "{\"title\":\"Total response time percentiles\",\"type\":\"line\",\"params\":{\"addTooltip\":true,\"addLegend\":true,\"legendPosition\":\"right\",\"showCircles\":true,\"interpolate\":\"linear\",\"scale\":\"linear\",\"drawLinesBetweenPoints\":true,\"radiusRatio\":9,\"times\":[],\"addTimeMarker\":false,\"defaultYExtents\":false,\"setYExtents\":false},\"aggs\":[{\"id\":\"1\",\"enabled\":true,\"type\":\"percentiles\",\"schema\":\"metric\",\"params\":{\"field\":\"app.total_time\",\"percents\":[50,90,95]}},{\"id\":\"2\",\"enabled\":true,\"type\":\"date_histogram\",\"schema\":\"segment\",\"params\":{\"field\":\"@timestamp\",\"interval\":\"auto\",\"customInterval\":\"2h\",\"min_doc_count\":1,\"extended_bounds\":{}}},{\"id\":\"3\",\"enabled\":true,\"type\":\"terms\",\"schema\":\"group\",\"params\":{\"field\":\"system.syslog.program\",\"size\":5,\"order\":\"desc\",\"orderBy\":\"_term\"}}],\"listeners\":{}}",
        "uiStateJSON": "{}",
        "description": "",
        "version": 1
      }
    }
  }
]
EOF
}


resource "opensearch_dashboard_object" "test_visualization_v7" {
  body = <<EOF
[
  {
    "_id": "response-time-percentile",
    "_source": {
      "type": "visualization",
      "visualization": {
        "title": "Total response time percentiles",
        "visState": "{\"title\":\"Total response time percentiles\",\"type\":\"line\",\"params\":{\"addTooltip\":true,\"addLegend\":true,\"legendPosition\":\"right\",\"showCircles\":true,\"interpolate\":\"linear\",\"scale\":\"linear\",\"drawLinesBetweenPoints\":true,\"radiusRatio\":9,\"times\":[],\"addTimeMarker\":false,\"defaultYExtents\":false,\"setYExtents\":false},\"aggs\":[{\"id\":\"1\",\"enabled\":true,\"type\":\"percentiles\",\"schema\":\"metric\",\"params\":{\"field\":\"app.total_time\",\"percents\":[50,90,95]}},{\"id\":\"2\",\"enabled\":true,\"type\":\"date_histogram\",\"schema\":\"segment\",\"params\":{\"field\":\"@timestamp\",\"interval\":\"auto\",\"customInterval\":\"2h\",\"min_doc_count\":1,\"extended_bounds\":{}}},{\"id\":\"3\",\"enabled\":true,\"type\":\"terms\",\"schema\":\"group\",\"params\":{\"field\":\"system.syslog.program\",\"size\":5,\"order\":\"desc\",\"orderBy\":\"_term\"}}],\"listeners\":{}}",
        "uiStateJSON": "{}",
        "description": "",
        "version": 1
      }
    }
  }
]
EOF
}

resource "opensearch_dashboard_object" "test_index_pattern_v6" {
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

resource "opensearch_dashboard_object" "test_index_pattern_v7" {
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
