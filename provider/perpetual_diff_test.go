package provider

import (
	"encoding/json"
	"testing"
)

// Shapes taken from real plans against OpenSearch 3 with Dashboards.

const authoredVisualization = `[{
  "_id": "visualization:vis00000-0000-4000-8000-000000000006",
  "_type": "doc",
  "_source": {
    "type": "visualization",
    "visualization": {
      "title": "Active detections",
      "visState": "{\"title\":\"Active detections\",\"type\":\"metric\",\"params\":{}}",
      "uiStateJSON": "{}",
      "kibanaSavedObjectMeta": {
        "searchSourceJSON": "{\"index\":\"alerts-alias\",\"query\":{\"query\":\"\",\"language\":\"kuery\"},\"filter\":[{\"meta\":{\"index\":\"alerts-alias\",\"key\":\"state\"},\"query\":{\"match_phrase\":{\"state\":\"ACTIVE\"}}},{\"meta\":{\"index\":\"alerts-alias\",\"key\":\"severity\"},\"query\":{\"match_phrase\":{\"severity\":\"1\"}}}]}"
      }
    }
  }
}]`

// What Dashboards stores after its saved-object migration ran, read back with
// OpenSearch 2+/3 semantics (no _type).
const migratedVisualization = `[{
  "_id": "visualization:vis00000-0000-4000-8000-000000000006",
  "_type": "",
  "_source": {
    "type": "visualization",
    "migrationVersion": {"visualization": "7.10.0"},
    "updated_at": "2026-10-16T08:00:00.000Z",
    "references": [
      {"id": "alerts-alias", "name": "kibanaSavedObjectMeta.searchSourceJSON.index", "type": "index-pattern"},
      {"id": "alerts-alias", "name": "kibanaSavedObjectMeta.searchSourceJSON.filter[0].meta.index", "type": "index-pattern"},
      {"id": "alerts-alias", "name": "kibanaSavedObjectMeta.searchSourceJSON.filter[1].meta.index", "type": "index-pattern"}
    ],
    "visualization": {
      "title": "Active detections",
      "visState": "{\"params\":{},\"title\":\"Active detections\",\"type\":\"metric\"}",
      "uiStateJSON": "{}",
      "kibanaSavedObjectMeta": {
        "searchSourceJSON": "{\"query\":{\"query\":\"\",\"language\":\"kuery\"},\"filter\":[{\"meta\":{\"key\":\"state\",\"indexRefName\":\"kibanaSavedObjectMeta.searchSourceJSON.filter[0].meta.index\"},\"query\":{\"match_phrase\":{\"state\":\"ACTIVE\"}}},{\"meta\":{\"key\":\"severity\",\"indexRefName\":\"kibanaSavedObjectMeta.searchSourceJSON.filter[1].meta.index\"},\"query\":{\"match_phrase\":{\"severity\":\"1\"}}}],\"indexRefName\":\"kibanaSavedObjectMeta.searchSourceJSON.index\"}"
      }
    }
  }
}]`

const authoredDashboard = `[{
  "_id": "dashboard:dash0000-0000-4000-8000-000000000001",
  "_type": "doc",
  "_source": {
    "type": "dashboard",
    "dashboard": {
      "title": "Alerting",
      "panelsJSON": "[{\"panelIndex\":\"1\",\"gridData\":{\"x\":0,\"y\":0,\"w\":24,\"h\":15,\"i\":\"1\"},\"id\":\"vis00000-0000-4000-8000-000000000006\",\"type\":\"visualization\",\"embeddableConfig\":{}}]",
      "optionsJSON": "{\"hidePanelTitles\":false}",
      "kibanaSavedObjectMeta": {"searchSourceJSON": "{\"query\":{\"query\":\"\",\"language\":\"kuery\"},\"filter\":[]}"}
    }
  }
}]`

const migratedDashboard = `[{
  "_id": "dashboard:dash0000-0000-4000-8000-000000000001",
  "_source": {
    "type": "dashboard",
    "migrationVersion": {"dashboard": "7.9.3"},
    "references": [{"name": "panel_0", "type": "visualization", "id": "vis00000-0000-4000-8000-000000000006"}],
    "dashboard": {
      "title": "Alerting",
      "panelsJSON": "[{\"embeddableConfig\":{},\"gridData\":{\"h\":15,\"i\":\"1\",\"w\":24,\"x\":0,\"y\":0},\"panelIndex\":\"1\",\"panelRefName\":\"panel_0\"}]",
      "optionsJSON": "{\"hidePanelTitles\":false}",
      "kibanaSavedObjectMeta": {"searchSourceJSON": "{\"filter\":[],\"query\":{\"language\":\"kuery\",\"query\":\"\"}}"}
    }
  }
}]`

func TestNormalizeDashboardObjectForState(t *testing.T) {
	tests := []struct {
		name           string
		config, server string
	}{
		{"_type doc vs empty", authoredVisualization, `[{"_id":"visualization:vis00000-0000-4000-8000-000000000006","_type":"","_source":` + mustSource(t, authoredVisualization) + `}]`},
		{"migrated visualization", authoredVisualization, migratedVisualization},
		{"migrated dashboard panels", authoredDashboard, migratedDashboard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Terraform compares the state string with StateFunc(config) byte for byte.
			got, want := normalizeDashboardObjectForState(tt.server), normalizeDashboardObjectForState(tt.config)
			if got != want {
				t.Errorf("normalized bodies differ\nserver: %s\nconfig: %s", got, want)
			}
		})
	}
}

func TestNormalizeDashboardObjectForStateKeepsRealChanges(t *testing.T) {
	changed := `[{"_id":"visualization:vis00000-0000-4000-8000-000000000006","_type":"doc","_source":{"type":"visualization","visualization":{"title":"Renamed","visState":"{}","uiStateJSON":"{}","kibanaSavedObjectMeta":{"searchSourceJSON":"{\"index\":\"other-alias\"}"}}}}]`
	if jsonEqual(t, normalizeDashboardObjectForState(changed), normalizeDashboardObjectForState(migratedVisualization)) {
		t.Fatal("a real title/index change must still show up")
	}
	// A reference that cannot be resolved keeps the references array so the diff stays visible.
	unresolved := `[{"_id":"x","_source":{"type":"visualization","references":[{"name":"unknown","type":"search","id":"s1"}],"visualization":{"title":"t"}}}]`
	var body []struct {
		Source map[string]any `json:"_source"`
	}
	if err := json.Unmarshal([]byte(normalizeDashboardObjectForState(unresolved)), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body[0].Source["references"]; !ok {
		t.Error("unresolved references must be kept")
	}
	if got := normalizeDashboardObjectForState("not json"); got != "not json" {
		t.Errorf("invalid JSON must pass through, got %q", got)
	}
}

func TestDiffSuppressChannelConfigurationConfigID(t *testing.T) {
	config := `{"config":{"name":"slack-test","description":"d","config_type":"slack","is_enabled":true,"slack":{"url":"https://hooks.slack.com/services/x"}}}`
	server := `{"config_id":"WVWr9p8BYu4z3eF8AWgy","created_time_ms":1,"last_updated_time_ms":2,"config":{"name":"slack-test","description":"d","config_type":"slack","is_enabled":true,"slack":{"url":"https://hooks.slack.com/services/x"}}}`
	if !diffSuppressChannelConfiguration("body", server, config, nil) {
		t.Error("config_id and timestamps returned by GET must not cause a diff")
	}
	changed := `{"config":{"name":"slack-test","description":"d","config_type":"slack","is_enabled":true,"slack":{"url":"https://hooks.slack.com/services/y"}}}`
	if diffSuppressChannelConfiguration("body", server, changed, nil) {
		t.Error("a changed webhook url must still diff")
	}
	// Read takes the resource id from config_id after normalizing the GET response.
	info := map[string]any{"config_id": "WVWr9p8BYu4z3eF8AWgy", "created_time_ms": 1}
	normalizeChannelConfiguration(info)
	if info["config_id"] != "WVWr9p8BYu4z3eF8AWgy" {
		t.Error("normalizeChannelConfiguration must keep config_id")
	}
}

func TestDiffSuppressIngestPipelineEmptyDescription(t *testing.T) {
	config := `{"processors":[{"set":{"field":"state","value":"ACTIVE","override":false}}]}`
	server := `{"description":"","processors":[{"set":{"field":"state","value":"ACTIVE","override":false}}]}`
	if !diffSuppressIngestPipeline("body", server, config, nil) {
		t.Error(`description "" returned by GET must not cause a diff`)
	}
	if diffSuppressIngestPipeline("body", `{"description":"x","processors":[]}`, `{"processors":[]}`, nil) {
		t.Error("a non-empty description must still diff")
	}
}

func TestDiffSuppressMonitorServerDefaults(t *testing.T) {
	config := `{"type":"monitor","monitor_type":"doc_level_monitor","name":"Doc-Level Monitor","enabled":true,
	  "schedule":{"period":{"interval":1,"unit":"MINUTES"}},
	  "inputs":[{"doc_level_input":{"description":"","indices":["alert-state"],"queries":[{"id":"q1","name":"new_active_alert","query":"state:ACTIVE","tags":["agg_alert"]}]}}],
	  "triggers":[]}`
	server := `{"type":"monitor","monitor_type":"doc_level_monitor","name":"Doc-Level Monitor","enabled":true,
	  "owner":"alerting","delete_query_index_in_every_run":false,"should_create_single_alert_for_findings":false,
	  "data_sources":{"alerts_history_index":".opendistro-alerting-alert-history-write","alerts_history_index_pattern":"<.opendistro-alerting-alert-history-{now/d}-1>","alerts_index":".opendistro-alerting-alerts","comments_index":".opensearch-alerting-comments-history-write","comments_index_pattern":"<.opensearch-alerting-comments-history-{now/d}-1>","findings_enabled":false,"findings_index":".opensearch-alerting-finding-history-write","findings_index_pattern":"<.opensearch-alerting-finding-history-{now/d}-1>","query_index":".opensearch-alerting-queries","query_index_mappings_by_type":{}},
	  "schedule":{"period":{"interval":1,"unit":"MINUTES"}},
	  "inputs":[{"doc_level_input":{"description":"","indices":["alert-state"],"fan_out_enabled":true,"queries":[{"id":"q1","name":"new_active_alert","query":"state:ACTIVE","tags":["agg_alert"],"fields":[],"query_field_names":[]}]}}],
	  "triggers":[]}`
	if !diffSuppressMonitor("body", server, config, nil) {
		t.Error("alerting plugin defaults must not cause a diff")
	}
	findings := `{"type":"monitor","monitor_type":"doc_level_monitor","name":"Doc-Level Monitor","enabled":true,"data_sources":{"findings_enabled":true},
	  "schedule":{"period":{"interval":1,"unit":"MINUTES"}},
	  "inputs":[{"doc_level_input":{"description":"","indices":["alert-state"],"queries":[{"id":"q1","name":"new_active_alert","query":"state:ACTIVE","tags":["agg_alert"]}]}}],
	  "triggers":[]}`
	if diffSuppressMonitor("body", server, findings, nil) {
		t.Error("a non-default data_sources must still diff")
	}
}

func mustSource(t *testing.T, body string) string {
	t.Helper()
	var b []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		t.Fatal(err)
	}
	return string(b[0]["_source"])
}

func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal([]byte(a), &x); err != nil {
		t.Fatalf("invalid JSON %q: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &y); err != nil {
		t.Fatalf("invalid JSON %q: %v", b, err)
	}
	ja, _ := json.Marshal(x)
	jb, _ := json.Marshal(y)
	return string(ja) == string(jb)
}
