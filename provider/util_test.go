package provider

import "testing"

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name   string
		result map[string]interface{}
		want   string
	}{
		{
			name: "error object with a reason",
			result: map[string]interface{}{
				"error": map[string]interface{}{
					"type":   "index_not_found_exception",
					"reason": "no such index [logs-000001]",
				},
			},
			want: "no such index [logs-000001]",
		},
		{
			// OpenSearch answers this way when the URI has no registered handler, which is what
			// a cluster running a version older than the endpoint returns. Before this was
			// handled the caller only ever saw "unknown error", discarding the one detail that
			// tells an operator their cluster is too old.
			name: "error as a plain string",
			result: map[string]interface{}{
				"error": "no handler found for uri [/_plugins/_example/thing] and method [GET]",
			},
			want: "no handler found for uri [/_plugins/_example/thing] and method [GET]",
		},
		{
			name:   "error as an empty string",
			result: map[string]interface{}{"error": ""},
			want:   "unknown error",
		},
		{
			name:   "error object without a reason",
			result: map[string]interface{}{"error": map[string]interface{}{"type": "exception"}},
			want:   "unknown error",
		},
		{
			name:   "no error field at all",
			result: map[string]interface{}{"acknowledged": true},
			want:   "unknown error",
		},
		{
			name:   "empty response",
			result: map[string]interface{}{},
			want:   "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractErrorMessage(tt.result); got != tt.want {
				t.Errorf("extractErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
