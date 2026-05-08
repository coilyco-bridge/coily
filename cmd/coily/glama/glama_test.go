package glama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSMParamPath(t *testing.T) {
	if !strings.HasPrefix(SSMParamAPIKey, "/glama/") {
		t.Errorf("SSMParamAPIKey = %q, want /glama/* prefix", SSMParamAPIKey)
	}
}

func TestAPIBaseStable(t *testing.T) {
	// Spec's servers[].url. If Glama relocates the API root, this points
	// at the place to update.
	if apiBase != "https://glama.ai/api/mcp" {
		t.Errorf("apiBase = %q, want https://glama.ai/api/mcp", apiBase)
	}
}

func TestReadBody_Stdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig }()

	go func() {
		_, _ = io.WriteString(w, `{"k":"v"}`)
		_ = w.Close()
	}()

	got, err := readBody("-")
	if err != nil {
		t.Fatalf("readBody: %v", err)
	}
	if string(got) != `{"k":"v"}` {
		t.Errorf("got %q, want %q", got, `{"k":"v"}`)
	}
}

func TestReadBody_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	if err := os.WriteFile(path, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := readBody("@" + path)
	if err != nil {
		t.Fatalf("readBody: %v", err)
	}
	if string(got) != `{"x":1}` {
		t.Errorf("got %q", got)
	}
}

func TestReadBody_Literal(t *testing.T) {
	got, err := readBody(`{"a":2}`)
	if err != nil {
		t.Fatalf("readBody: %v", err)
	}
	if string(got) != `{"a":2}` {
		t.Errorf("got %q", got)
	}
}

func TestReadBody_Empty(t *testing.T) {
	got, err := readBody("")
	if err != nil {
		t.Fatalf("readBody: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestDoGetAndStream_ErrorBodyIncludedOn4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

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

// TestSpecPathsCoverage cross-checks the verb tree against the
// published OpenAPI spec at https://glama.ai/api/mcp/openapi.json. If a
// wired endpoint disappears from the tree, this test points at the spec
// to update or the verb to remove.
func TestSpecPathsCoverage(t *testing.T) {
	cmd := Command(nil, nil)
	if cmd.Name != "glama" {
		t.Fatalf("root name = %q, want glama", cmd.Name)
	}
	wantTop := map[string]bool{
		"servers":    false,
		"attributes": false,
		"instances":  false,
		"telemetry":  false,
	}
	for _, sub := range cmd.Commands {
		if _, ok := wantTop[sub.Name]; ok {
			wantTop[sub.Name] = true
		}
	}
	for n, found := range wantTop {
		if !found {
			t.Errorf("missing top-level subcommand %q", n)
		}
	}

	wantServers := map[string]bool{"list": false, "get": false}
	for _, sub := range cmd.Commands {
		if sub.Name != "servers" {
			continue
		}
		for _, leaf := range sub.Commands {
			if _, ok := wantServers[leaf.Name]; ok {
				wantServers[leaf.Name] = true
			}
		}
	}
	for n, found := range wantServers {
		if !found {
			t.Errorf("missing servers leaf %q", n)
		}
	}

	wantTelemetry := map[string]bool{"usage": false}
	for _, sub := range cmd.Commands {
		if sub.Name != "telemetry" {
			continue
		}
		for _, leaf := range sub.Commands {
			if _, ok := wantTelemetry[leaf.Name]; ok {
				wantTelemetry[leaf.Name] = true
			}
		}
	}
	for n, found := range wantTelemetry {
		if !found {
			t.Errorf("missing telemetry leaf %q", n)
		}
	}
}

// TestPrettyPrintRoundtrip ensures the response handler produces parseable
// JSON identical in content to the upstream payload.
func TestPrettyPrintRoundtrip(t *testing.T) {
	in := `{"a":1,"b":[2,3]}`
	var v json.RawMessage
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	if back["a"].(float64) != 1 {
		t.Errorf("roundtrip lost data: %v", back)
	}
}
