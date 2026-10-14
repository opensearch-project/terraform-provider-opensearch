package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
)

// fakePredictServer simulates the OpenSearch _predict endpoint for unit-testing
// waitForModelPredictReady. Each POST against /_plugins/_ml/models/<id>/_predict
// returns the next response from a scripted FIFO queue; once exhausted, the last
// entry repeats. If no responses are scripted, it defaults to an immediate success
// (inference_results present).
type fakePredictServer struct {
	t                      *testing.T
	server                 *httptest.Server
	predictCalls           atomic.Int32
	lastPredictRequestBody atomic.Value
	responses              []fakePredictResponse
}

type fakePredictResponse struct {
	statusCode int
	body       string
}

var (
	predictSuccess = fakePredictResponse{http.StatusOK, `{"inference_results": [{"output": []}]}`}
	predictEvicted = fakePredictResponse{http.StatusOK, `{"error": "model not deployed in this node"}`}
	predictHTTP503 = fakePredictResponse{http.StatusServiceUnavailable, `{"error": "ML memory circuit breaker triggered"}`}
)

func newFakePredictServer(t *testing.T, responses []fakePredictResponse) *fakePredictServer {
	t.Helper()
	f := &fakePredictServer{t: t, responses: responses}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakePredictServer) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_predict") {
		raw, _ := io.ReadAll(r.Body)
		f.lastPredictRequestBody.Store(string(raw))

		idx := int(f.predictCalls.Add(1) - 1)

		resp := predictSuccess // default when no scripted responses remain
		if len(f.responses) > 0 {
			if idx < len(f.responses) {
				resp = f.responses[idx]
			} else {
				resp = f.responses[len(f.responses)-1] // repeat last
			}
		}

		w.WriteHeader(resp.statusCode)
		_, _ = w.Write([]byte(resp.body))
		return
	}

	f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNotFound)
}

func (f *fakePredictServer) conf(t *testing.T) *ProviderConf {
	t.Helper()
	client, err := opensearch.NewClient(opensearch.Config{Addresses: []string{f.server.URL}})
	if err != nil {
		t.Fatalf("failed to create opensearch client: %v", err)
	}
	return &ProviderConf{rawUrl: f.server.URL, osClient: client}
}

func shortenPredictProbeTimings(t *testing.T, timeout time.Duration) {
	t.Helper()
	originalTimeout, originalInterval := maxPredictProbeWait, predictProbePollInterval
	maxPredictProbeWait = timeout
	predictProbePollInterval = time.Millisecond
	t.Cleanup(func() {
		maxPredictProbeWait = originalTimeout
		predictProbePollInterval = originalInterval
	})
}

func TestWaitForModelPredictReady_ImmediateSuccess(t *testing.T) {
	shortenPredictProbeTimings(t, 5*time.Second)
	fake := newFakePredictServer(t, []fakePredictResponse{predictSuccess})

	err := waitForModelPredictReady(context.Background(), fake.conf(t), "m1", defaultPredictProbeBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := fake.predictCalls.Load(), int32(1); got != want {
		t.Errorf("predict calls: got %d, want %d", got, want)
	}
}

// A custom probe body must be forwarded verbatim to the _predict endpoint.
func TestWaitForModelPredictReady_CustomProbeBody(t *testing.T) {
	shortenPredictProbeTimings(t, 5*time.Second)
	fake := newFakePredictServer(t, []fakePredictResponse{predictSuccess})

	customBody := `{"text_docs": ["healthcheck"]}`
	err := waitForModelPredictReady(context.Background(), fake.conf(t), "m1", customBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := fake.lastPredictRequestBody.Load().(string); got != customBody {
		t.Errorf("probe body: got %q, want %q", got, customBody)
	}
}

// The "model evicted from memory" case: _predict returns HTTP 200 with an error body
// and no 'inference_results' key. The probe must retry until inference_results appears.
func TestWaitForModelPredictReady_RetriesOnEvictedModel(t *testing.T) {
	shortenPredictProbeTimings(t, 5*time.Second)
	fake := newFakePredictServer(t, []fakePredictResponse{
		predictEvicted,
		predictEvicted,
		predictSuccess,
	})

	err := waitForModelPredictReady(context.Background(), fake.conf(t), "m1", defaultPredictProbeBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := fake.predictCalls.Load(), int32(3); got != want {
		t.Errorf("predict calls: got %d, want %d", got, want)
	}
}

// An HTTP error (e.g. circuit breaker returning 503) must also be retried.
func TestWaitForModelPredictReady_RetriesOnHTTPError(t *testing.T) {
	shortenPredictProbeTimings(t, 5*time.Second)
	fake := newFakePredictServer(t, []fakePredictResponse{
		predictHTTP503,
		predictSuccess,
	})

	err := waitForModelPredictReady(context.Background(), fake.conf(t), "m1", defaultPredictProbeBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := fake.predictCalls.Load(), int32(2); got != want {
		t.Errorf("predict calls: got %d, want %d", got, want)
	}
}

// When the probe never sees 'inference_results' before the deadline it must return a descriptive timeout error.
func TestWaitForModelPredictReady_TimesOut(t *testing.T) {
	shortenPredictProbeTimings(t, 50*time.Millisecond)
	fake := newFakePredictServer(t, []fakePredictResponse{predictEvicted}) // repeats forever

	err := waitForModelPredictReady(context.Background(), fake.conf(t), "m1", defaultPredictProbeBody)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout waiting for ML Model predict readiness") {
		t.Errorf("expected error to mention timeout, got: %v", err)
	}
	if !strings.Contains(err.Error(), "m1") {
		t.Errorf("expected error to include ML Model ID, got: %v", err)
	}
}

// A canceled context must abort the probe immediately regardless of model state.
func TestWaitForModelPredictReady_ContextCancelled(t *testing.T) {
	shortenPredictProbeTimings(t, 5*time.Second)
	fake := newFakePredictServer(t, []fakePredictResponse{predictEvicted}) // repeats forever

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForModelPredictReady(ctx, fake.conf(t), "m1", defaultPredictProbeBody)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if !strings.Contains(err.Error(), "context cancelled while waiting") {
		t.Errorf("expected error to mention context, got: %v", err)
	}
}
