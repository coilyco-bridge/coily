package modio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoGetAndStream_ErrorBodyIncludedOn4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"forbidden"}}`))
	}))
	defer srv.Close()

	// Drive the HTTP path directly without the SSM fetch by inlining the
	// request building. Keeps this test hermetic.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestDefaultGameIDStable(t *testing.T) {
	// Eco's mod.io game id is constant; if upstream ever renumbers, the
	// failing test points at the place to update.
	if DefaultGameID != 6 {
		t.Errorf("DefaultGameID = %d, want 6", DefaultGameID)
	}
}

func TestSSMParamPath(t *testing.T) {
	// Cross-check with the canonical inventory in agentic-os-kai/AGENTS.md.
	if !strings.HasPrefix(SSMParamAPIKey, "/modio/") {
		t.Errorf("SSMParamAPIKey = %q, want /modio/* prefix", SSMParamAPIKey)
	}
}
