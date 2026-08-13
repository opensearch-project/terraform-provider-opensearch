package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDashboardsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dash/api/workspaces" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get(dashboardsXsrfHeader) != "true" {
			t.Fatalf("missing %s header", dashboardsXsrfHeader)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "admin" || password != "secret" {
			t.Fatalf("unexpected basic auth: %s/%s", username, password)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"workspace-1"}}`))
	}))
	defer server.Close()

	dashboardsURL, err := url.Parse(server.URL + "/dash")
	if err != nil {
		t.Fatal(err)
	}
	conf := &ProviderConf{
		dashboardsUrl:       dashboardsURL.String(),
		parsedDashboardsUrl: dashboardsURL,
		username:            "admin",
		password:            "secret",
	}

	var response dashboardWorkspaceCreateResponse
	if err := dashboardsRequest(conf, http.MethodPost, "/api/workspaces", map[string]string{"name": "logs"}, &response); err != nil {
		t.Fatal(err)
	}
	if !response.Success || response.Result.ID != "workspace-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestDashboardsRequestRequiresDashboardsURL(t *testing.T) {
	err := dashboardsRequest(&ProviderConf{}, http.MethodGet, "/api/workspaces/test", nil, nil)
	if err == nil {
		t.Fatal("expected dashboards_url error")
	}
}

func TestDashboardsRequestReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	dashboardsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	conf := &ProviderConf{
		dashboardsUrl:       dashboardsURL.String(),
		parsedDashboardsUrl: dashboardsURL,
	}

	err = dashboardsRequest(conf, http.MethodGet, "/api/workspaces/missing", nil, nil)
	if err != errDashboardsNotFound {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestDashboardWorkspaceRequestBody(t *testing.T) {
	resource := resourceOpensearchDashboardWorkspace()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name":        "Logs",
		"description": "Production logs",
		"features":    []interface{}{"use-case-observability"},
		"permissions": `{"library_write":["admin"]}`,
	})

	body, err := dashboardWorkspaceRequestBody(data)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"attributes":{"name":"Logs","description":"Production logs","features":["use-case-observability"]},"permissions":{"library_write":["admin"]}}`
	if string(encoded) != expected {
		t.Fatalf("unexpected body: %s", string(encoded))
	}
}

func TestDashboardWorkspaceObjectsRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/_associate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		expected := `{"workspaceId":"workspace-1","savedObjects":[{"type":"index-pattern","id":"logs-*"}]}`
		if string(body) != expected {
			t.Fatalf("unexpected request body: %s", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"logs-*"}]}`))
	}))
	defer server.Close()

	dashboardsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	conf := &ProviderConf{
		dashboardsUrl:       dashboardsURL.String(),
		parsedDashboardsUrl: dashboardsURL,
	}

	resource := resourceOpensearchDashboardWorkspaceObjects()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"workspace_id": "workspace-1",
		"saved_object": []interface{}{
			map[string]interface{}{
				"type": "index-pattern",
				"id":   "logs-*",
			},
		},
	})

	if err := associateDashboardWorkspaceObjects(data, conf, "/api/workspaces/_associate"); err != nil {
		t.Fatal(err)
	}
}
