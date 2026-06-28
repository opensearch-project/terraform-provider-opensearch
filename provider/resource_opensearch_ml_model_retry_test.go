package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
)

// fakeMLModelServer simulates the OpenSearch ML Model API for unit-testing the
// deployment retry path. Each GET against /_plugins/_ml/models/<id> returns the
// next state from a scripted FIFO queue (DEPLOYED if the queue is exhausted),
// and POSTs to /_deploy and /_undeploy are counted but otherwise no-ops.
type fakeMLModelServer struct {
	t             *testing.T
	server        *httptest.Server
	deployCalls   atomic.Int32
	undeployCalls atomic.Int32
	getCalls      atomic.Int32
	modelStates   []string
}

func newFakeMLModelServer(t *testing.T, modelStates []string) *fakeMLModelServer {
	t.Helper()
	f := &fakeMLModelServer{t: t, modelStates: modelStates}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeMLModelServer) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_deploy"):
		f.deployCalls.Add(1)
		_, _ = w.Write([]byte(`{"task_id":"t1"}`))
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_undeploy"):
		f.undeployCalls.Add(1)
		_, _ = w.Write([]byte(`{}`))
	case r.Method == http.MethodGet:
		idx := int(f.getCalls.Add(1) - 1)
		state := "DEPLOYED"
		if idx < len(f.modelStates) {
			state = f.modelStates[idx]
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"model_state": state})
	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeMLModelServer) conf(t *testing.T) *ProviderConf {
	t.Helper()
	client, err := opensearch.NewClient(opensearch.Config{Addresses: []string{f.server.URL}})
	if err != nil {
		t.Fatalf("failed to create opensearch client: %v", err)
	}
	return &ProviderConf{rawUrl: f.server.URL, osClient: client}
}

func TestDeployMLModel_HappyPath(t *testing.T) {
	fake := newFakeMLModelServer(t, []string{"DEPLOYED"})

	if err := deployMLModel(context.Background(), fake.conf(t), "m1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := fake.deployCalls.Load(), int32(1); got != want {
		t.Errorf("deploy calls: got %d, want %d", got, want)
	}
	if got, want := fake.undeployCalls.Load(), int32(0); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
}

func TestDeployMLModel_PartiallyDeployedTriggersUndeployAndRedeploy(t *testing.T) {
	// States served in order by the fake GET handler:
	//   [0] PARTIALLY_DEPLOYED - first deployAndWait returns this, triggering the retry path
	//   [1] UNDEPLOYED         - undeployMLModel's post-undeploy poll sees UNDEPLOYED and returns
	//   [2] DEPLOYED           - second deployAndWait returns DEPLOYED (success)
	fake := newFakeMLModelServer(t, []string{"PARTIALLY_DEPLOYED", "UNDEPLOYED", "DEPLOYED"})

	if err := deployMLModel(context.Background(), fake.conf(t), "m1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := fake.deployCalls.Load(), int32(2); got != want {
		t.Errorf("deploy calls: got %d, want %d", got, want)
	}
	if got, want := fake.undeployCalls.Load(), int32(1); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
}

func TestDeployMLModel_UndeployedTriggersRedeployOnly(t *testing.T) {
	fake := newFakeMLModelServer(t, []string{"UNDEPLOYED", "DEPLOYED"})

	if err := deployMLModel(context.Background(), fake.conf(t), "m1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := fake.deployCalls.Load(), int32(2); got != want {
		t.Errorf("deploy calls: got %d, want %d", got, want)
	}
	if got, want := fake.undeployCalls.Load(), int32(0); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
}

func TestDeployMLModel_FailsAfterSingleRetry(t *testing.T) {
	// States served in order:
	//   [0] PARTIALLY_DEPLOYED - first deployAndWait returns this, triggering the retry path
	//   [1] UNDEPLOYED         - undeployMLModel's post-undeploy poll sees UNDEPLOYED and returns
	//   [2] PARTIALLY_DEPLOYED - second deployAndWait still not DEPLOYED → error
	fake := newFakeMLModelServer(t, []string{"PARTIALLY_DEPLOYED", "UNDEPLOYED", "PARTIALLY_DEPLOYED"})

	err := deployMLModel(context.Background(), fake.conf(t), "m1")
	if err == nil {
		t.Fatal("expected error after retry exhausted, got nil")
	}
	if !strings.Contains(err.Error(), "PARTIALLY_DEPLOYED") {
		t.Errorf("expected error to mention final state, got: %v", err)
	}
	if got, want := fake.deployCalls.Load(), int32(2); got != want {
		t.Errorf("deploy calls: got %d, want %d", got, want)
	}
	if got, want := fake.undeployCalls.Load(), int32(1); got != want {
		t.Errorf("undeploy calls: got %d, want %d", got, want)
	}
}
