package shell

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// embeddedManifest holds the in-tree JSON describing where to fetch each
// pinned tool binary and what its sha256 must be. Built into the coily
// binary at compile time. PRODUCTION USE NOTE: the placeholder SHAs in
// tools.json must be filled in by the .github/workflows/release-tools.yml
// workflow before any fetch will succeed. Until then, FetchingResolver
// errors at runtime when a real fetch is attempted.
//
//go:embed tools.json
var embeddedManifest []byte

// PlatformEntry pins one (tool, goos, goarch) to a fetch URL and a sha256
// the downloaded bytes must match. URL is where coily fetches from. The
// upstream_url field is metadata so a human (or the release workflow) can
// trace where the binary originally came from.
type PlatformEntry struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	UpstreamURL string `json:"upstream_url"`
	SHA256      string `json:"sha256"`
}

// ToolManifest is the structure of pkg/shell/tools.json.
type ToolManifest struct {
	ReleaseTag     string `json:"release_tag"`
	ReleaseURLBase string `json:"release_url_base"`
	// Tools maps tool name -> "goos/goarch" -> PlatformEntry.
	Tools map[string]map[string]PlatformEntry `json:"tools"`
}

// LoadEmbeddedManifest parses the in-tree tools.json that was embedded
// into the coily binary at build time.
func LoadEmbeddedManifest() (*ToolManifest, error) {
	return ParseManifest(embeddedManifest)
}

// ParseManifest parses a tools.json byte slice. Exported so tests can
// build a fixture manifest pointing at an httptest.Server.
func ParseManifest(b []byte) (*ToolManifest, error) {
	var m ToolManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("shell: parse tools manifest: %w", err)
	}
	if m.Tools == nil {
		return nil, errors.New("shell: tools manifest has no tools map")
	}
	return &m, nil
}

// Lookup returns the PlatformEntry for (tool, goos, goarch) or an error if
// the manifest doesn't pin that combination.
func (m *ToolManifest) Lookup(tool, goos, goarch string) (PlatformEntry, error) {
	platforms, ok := m.Tools[tool]
	if !ok {
		return PlatformEntry{}, fmt.Errorf("shell: tool %q not pinned in manifest", tool)
	}
	key := goos + "/" + goarch
	e, ok := platforms[key]
	if !ok {
		return PlatformEntry{}, fmt.Errorf("shell: tool %q has no entry for %s", tool, key)
	}
	if e.SHA256 == "" {
		return PlatformEntry{}, fmt.Errorf("shell: tool %q on %s has empty sha256", tool, key)
	}
	if strings.HasPrefix(e.SHA256, "PLACEHOLDER_") {
		return PlatformEntry{}, fmt.Errorf("shell: tool %q on %s still has placeholder sha256 (run release-tools workflow)", tool, key)
	}
	return e, nil
}

// HTTPDoer is the subset of *http.Client that FetchingResolver needs.
// Tests inject httptest.NewServer's client to avoid real network calls.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// FetchingResolver resolves binary names to absolute paths by consulting
// the embedded ToolManifest, fetching from the pinned URL on cache miss,
// verifying the sha256, and caching under CacheDir. It is the production
// replacement for PathResolver. PATH is intentionally not consulted:
// the threat model says an agent who swaps /usr/local/bin/aws should
// still be ignored.
type FetchingResolver struct {
	// Manifest is the source of truth for (URL, SHA256) per (tool, platform).
	Manifest *ToolManifest
	// CacheDir is the base directory for cached binaries. Layout is
	// CacheDir/<sha256>/<tool>. Defaults to ~/.cache/coily/bin when "".
	CacheDir string
	// GOOS / GOARCH default to runtime.GOOS / runtime.GOARCH when "".
	// Tests override to assert cross-platform behavior.
	GOOS, GOARCH string
	// HTTPClient defaults to a 5-minute-timeout http.Client when nil.
	HTTPClient HTTPDoer

	mu sync.Mutex
}

// NewFetchingResolver builds a FetchingResolver from the embedded manifest.
// Returns an error if the manifest fails to parse.
func NewFetchingResolver() (*FetchingResolver, error) {
	m, err := LoadEmbeddedManifest()
	if err != nil {
		return nil, err
	}
	return &FetchingResolver{Manifest: m}, nil
}

// defaultCacheDir returns ~/.cache/coily/bin or empty if HOME is unset.
func defaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "coily", "bin")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "coily", "bin")
}

func (f *FetchingResolver) goos() string {
	if f.GOOS != "" {
		return f.GOOS
	}
	return runtime.GOOS
}

func (f *FetchingResolver) goarch() string {
	if f.GOARCH != "" {
		return f.GOARCH
	}
	return runtime.GOARCH
}

func (f *FetchingResolver) cacheDir() string {
	if f.CacheDir != "" {
		return f.CacheDir
	}
	return defaultCacheDir()
}

func (f *FetchingResolver) httpClient() HTTPDoer {
	if f.HTTPClient != nil {
		return f.HTTPClient
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

// Resolve returns the absolute path to the cached binary for `bin`. On
// cache miss, fetches from the manifest URL, verifies sha256, writes to
// cache with mode 0700. On checksum mismatch, removes the corrupted file
// and returns an error. Safe for concurrent calls within one process.
func (f *FetchingResolver) Resolve(bin string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entry, err := f.Manifest.Lookup(bin, f.goos(), f.goarch())
	if err != nil {
		return "", err
	}
	cache := f.cacheDir()
	if cache == "" {
		return "", errors.New("shell: cache dir unresolved (HOME and XDG_CACHE_HOME both empty)")
	}
	dir := filepath.Join(cache, entry.SHA256)
	path := filepath.Join(dir, bin)

	// Cache hit: file exists and checksum still matches. We re-verify on
	// every Resolve so a tampered cache file is detected before exec.
	if _, err := os.Stat(path); err == nil {
		ok, vErr := verifySHA256(path, entry.SHA256)
		if vErr == nil && ok {
			return path, nil
		}
		// Corrupted or unreadable: nuke it and re-fetch.
		_ = os.Remove(path)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("shell: mkdir cache: %w", err)
	}
	if err := f.fetchAndVerify(entry.URL, entry.SHA256, path); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", fmt.Errorf("shell: chmod cache: %w", err)
	}
	return path, nil
}

// fetchAndVerify downloads url to dst and errors if the sha256 of the
// bytes doesn't match wantHex. Writes via a temp file to avoid leaving a
// partial binary in the cache on interrupt.
func (f *FetchingResolver) fetchAndVerify(url, wantHex, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("shell: build fetch request: %w", err)
	}
	resp, err := f.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("shell: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shell: fetch %s: status %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".coily-tool-*")
	if err != nil {
		return fmt.Errorf("shell: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if anything below fails.
	defer func() { _ = os.Remove(tmpName) }()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("shell: read body: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("shell: close tempfile: %w", err)
	}
	gotHex := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(gotHex, wantHex) {
		return fmt.Errorf("shell: sha256 mismatch for %s: got %s, want %s", url, gotHex, wantHex)
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return fmt.Errorf("shell: install cache file: %w", err)
	}
	return nil
}

// verifySHA256 returns (true, nil) iff the file at path hashes to wantHex.
func verifySHA256(path, wantHex string) (bool, error) {
	fp, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer fp.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fp); err != nil {
		return false, err
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), wantHex), nil
}

// AsResolverFunc adapts a FetchingResolver to the function-typed Resolver
// the Runner consumes. Wired in cmd/coily/runtime.go.
func (f *FetchingResolver) AsResolverFunc() Resolver {
	return f.Resolve
}
