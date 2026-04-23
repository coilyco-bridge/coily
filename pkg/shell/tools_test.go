package shell_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/shell"
	"gopkg.in/yaml.v3"
)

// fixtureManifest builds a single-entry ToolManifest pointing at the given
// URL with the given sha256. Lets each test focus on one behavior. Tests
// always use linux/amd64 as the platform key; the FetchingResolver under
// test sets GOOS/GOARCH explicitly.
func fixtureManifest(tool, url, sha string) *shell.ToolManifest {
	return &shell.ToolManifest{
		ReleaseTag: "tools-latest",
		Tools: map[string]map[string]shell.PlatformEntry{
			tool: {
				"linux/amd64": {
					Version: "test-1.0",
					URL:     url,
					SHA256:  sha,
				},
			},
		},
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestFetchingResolver_CacheMissThenFetch(t *testing.T) {
	body := []byte("#!/bin/sh\necho stub-aws\n")
	want := sha256Hex(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	cache := t.TempDir()
	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("aws", srv.URL+"/aws", want),
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	path, err := r.Resolve("aws")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(path, cache) {
		t.Errorf("path %q not under cache %q", path, cache)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cached file: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("cached body = %q, want %q", got, body)
	}
}

func TestFetchingResolver_CacheHitNoFetch(t *testing.T) {
	body := []byte("cached-content")
	want := sha256Hex(body)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Write(body)
	}))
	defer srv.Close()

	cache := t.TempDir()
	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("kubectl", srv.URL+"/kubectl", want),
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	if _, err := r.Resolve("kubectl"); err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if hits != 1 {
		t.Fatalf("first call hits = %d, want 1", hits)
	}
	if _, err := r.Resolve("kubectl"); err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if hits != 1 {
		t.Errorf("second call should be cache hit, hits = %d", hits)
	}
}

func TestFetchingResolver_ChecksumMismatchErrors(t *testing.T) {
	body := []byte("not-what-the-manifest-says")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	cache := t.TempDir()
	wrongSHA := sha256Hex([]byte("something-else-entirely"))
	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("gh", srv.URL+"/gh", wrongSHA),
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("gh")
	if err == nil {
		t.Fatal("expected sha256 mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("err = %v, want sha256 mismatch", err)
	}
	// And nothing should be cached on disk.
	entries, _ := os.ReadDir(filepath.Join(cache, wrongSHA))
	for _, e := range entries {
		if e.Name() == "gh" {
			t.Error("cached corrupt binary should have been removed")
		}
	}
}

func TestFetchingResolver_CorruptedCacheRefetches(t *testing.T) {
	body := []byte("real-binary-bytes")
	want := sha256Hex(body)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Write(body)
	}))
	defer srv.Close()

	cache := t.TempDir()
	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("aws", srv.URL+"/aws", want),
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	path, err := r.Resolve("aws")
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if hits != 1 {
		t.Fatalf("first call hits = %d", hits)
	}
	// Corrupt the cached file by overwriting it.
	if err := os.WriteFile(path, []byte("tampered"), 0o700); err != nil {
		t.Fatalf("write tamper: %v", err)
	}
	if _, err := r.Resolve("aws"); err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if hits != 2 {
		t.Errorf("corrupted cache should re-fetch, hits = %d, want 2", hits)
	}
	// And the file should now hash correctly again.
	got, _ := os.ReadFile(path)
	if string(got) != string(body) {
		t.Errorf("post-refetch body = %q, want %q", got, body)
	}
}

func TestFetchingResolver_UnpinnedToolErrors(t *testing.T) {
	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("aws", "http://nope", sha256Hex([]byte("x"))),
		CacheDir: t.TempDir(),
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("kubectl")
	if err == nil {
		t.Fatal("expected error for unpinned tool")
	}
	if !strings.Contains(err.Error(), "not pinned") {
		t.Errorf("err = %v, want 'not pinned'", err)
	}
}

func TestFetchingResolver_PlatformMissingErrors(t *testing.T) {
	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("aws", "http://nope", sha256Hex([]byte("x"))),
		CacheDir: t.TempDir(),
		GOOS:     "windows",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("aws")
	if err == nil {
		t.Fatal("expected error for missing platform entry")
	}
	if !strings.Contains(err.Error(), "no entry for") {
		t.Errorf("err = %v, want 'no entry for'", err)
	}
}

func TestFetchingResolver_PlaceholderSHARefuses(t *testing.T) {
	r := &shell.FetchingResolver{
		Manifest: &shell.ToolManifest{
			Tools: map[string]map[string]shell.PlatformEntry{
				"aws": {"linux/amd64": {URL: "http://nope", SHA256: "PLACEHOLDER_AWS_LINUX_AMD64"}},
			},
		},
		CacheDir: t.TempDir(),
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("aws")
	if err == nil {
		t.Fatal("expected placeholder sha to refuse")
	}
	if !strings.Contains(err.Error(), "placeholder sha256") {
		t.Errorf("err = %v, want placeholder refusal", err)
	}
}

func TestFetchingResolver_HTTPErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	r := &shell.FetchingResolver{
		Manifest: fixtureManifest("aws", srv.URL+"/aws", sha256Hex([]byte("x"))),
		CacheDir: t.TempDir(),
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("aws")
	if err == nil {
		t.Fatal("expected error on 404")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("err = %v, want status 404", err)
	}
}

func TestEmbeddedManifest_ParsesAndCoversCoreTools(t *testing.T) {
	// The in-tree tools.yaml must always be valid YAML and must always
	// pin aws / kubectl / gh on the four platforms coily targets. SHAs
	// can be placeholders pre-release; the parse must still succeed.
	m, err := shell.LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("LoadEmbeddedManifest: %v", err)
	}
	for _, tool := range []string{"aws", "kubectl", "gh"} {
		platforms, ok := m.Tools[tool]
		if !ok {
			t.Errorf("tool %s missing from embedded manifest", tool)
			continue
		}
		for _, plat := range []string{"darwin/arm64", "darwin/amd64", "linux/amd64", "linux/arm64"} {
			if _, ok := platforms[plat]; !ok {
				t.Errorf("tool %s missing platform %s", tool, plat)
			}
		}
	}
}

// buildTarGz returns a tar.gz byte slice containing the given files.
// Used to synthesize archive fixtures for the archive-resolver tests
// without depending on any on-disk tooling. Files is an ordered list
// of (relative path, mode, contents). Directories are created
// implicitly from file parents.
func buildTarGz(t *testing.T, files []struct {
	Name string
	Mode int64
	Data []byte
}) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	seenDirs := map[string]bool{}
	for _, f := range files {
		dir := filepath.Dir(f.Name)
		if dir != "." && !seenDirs[dir] {
			if err := tw.WriteHeader(&tar.Header{
				Name:     dir + "/",
				Mode:     0o700,
				Typeflag: tar.TypeDir,
			}); err != nil {
				t.Fatalf("tar write dir: %v", err)
			}
			seenDirs[dir] = true
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:     f.Name,
			Mode:     f.Mode,
			Size:     int64(len(f.Data)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("tar write file: %v", err)
		}
		if _, err := tw.Write(f.Data); err != nil {
			t.Fatalf("tar write body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestFetchingResolver_ArchiveExtractsBundle(t *testing.T) {
	// Simulates aws-cli's dist/ bundle: a launcher plus a sibling
	// shared lib. The Resolve result must point at dist/aws and the
	// sibling Python file must land next to it so dlopen() works.
	archive := buildTarGz(t, []struct {
		Name string
		Mode int64
		Data []byte
	}{
		{"dist/aws", 0o755, []byte("#!/bin/sh\nexec ./Python aws.py \"$@\"\n")},
		{"dist/Python", 0o755, []byte("fake-python-shared-lib")},
		{"dist/aws.py", 0o644, []byte("print('hello')")},
	})
	want := sha256Hex(archive)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	cache := t.TempDir()
	r := &shell.FetchingResolver{
		Manifest: &shell.ToolManifest{
			Tools: map[string]map[string]shell.PlatformEntry{
				"aws": {
					"linux/amd64": {
						Version: "test-1.0",
						URL:     srv.URL + "/aws.tar.gz",
						SHA256:  want,
						Archive: "tar.gz",
						Entry:   "dist/aws",
					},
				},
			},
		},
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	path, err := r.Resolve("aws")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if filepath.Base(path) != "aws" || filepath.Base(filepath.Dir(path)) != "dist" {
		t.Errorf("Resolve path = %q, want .../dist/aws", path)
	}
	// Entry content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if !strings.Contains(string(got), "exec ./Python") {
		t.Errorf("entry content = %q, want aws launcher stub", got)
	}
	// Sibling shared lib must be present so the launcher can dlopen it.
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "Python")); err != nil {
		t.Errorf("sibling Python missing: %v", err)
	}
	// Seal marker must exist at the archive root.
	if _, err := os.Stat(filepath.Join(cache, want, ".coily-archive-sha256")); err != nil {
		t.Errorf("seal marker missing: %v", err)
	}

	// Second Resolve: no new HTTP hit required. Use a fresh server that
	// would fail if called, to prove cache hit.
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer fail.Close()
	r.Manifest.Tools["aws"]["linux/amd64"] = shell.PlatformEntry{
		Version: "test-1.0",
		URL:     fail.URL + "/aws.tar.gz",
		SHA256:  want,
		Archive: "tar.gz",
		Entry:   "dist/aws",
	}
	if _, err := r.Resolve("aws"); err != nil {
		t.Errorf("second Resolve (cache hit) failed: %v", err)
	}
}

func TestFetchingResolver_ArchiveChecksumMismatchCleansUp(t *testing.T) {
	archive := buildTarGz(t, []struct {
		Name string
		Mode int64
		Data []byte
	}{
		{"dist/aws", 0o755, []byte("launcher")},
	})
	wrongSHA := sha256Hex([]byte("some other payload"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	cache := t.TempDir()
	r := &shell.FetchingResolver{
		Manifest: &shell.ToolManifest{
			Tools: map[string]map[string]shell.PlatformEntry{
				"aws": {
					"linux/amd64": {
						Version: "test-1.0",
						URL:     srv.URL + "/aws.tar.gz",
						SHA256:  wrongSHA,
						Archive: "tar.gz",
						Entry:   "dist/aws",
					},
				},
			},
		},
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("aws")
	if err == nil {
		t.Fatal("expected sha256 mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("err = %v, want sha256 mismatch", err)
	}
	// No archive dir should remain after a failed extraction.
	if _, err := os.Stat(filepath.Join(cache, wrongSHA)); err == nil {
		t.Errorf("archive dir exists after failed extraction")
	}
}

func TestFetchingResolver_ArchiveRejectsUnsafePaths(t *testing.T) {
	// A tarball whose entry path escapes the extract root must be
	// rejected without writing any files outside the cache.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("pwned")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "../../../etc/passwd",
		Mode:     0o644,
		Size:     int64(len(body)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("tar body: %v", err)
	}
	tw.Close()
	gz.Close()
	archive := buf.Bytes()
	want := sha256Hex(archive)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(archive)
	}))
	defer srv.Close()

	cache := t.TempDir()
	r := &shell.FetchingResolver{
		Manifest: &shell.ToolManifest{
			Tools: map[string]map[string]shell.PlatformEntry{
				"aws": {
					"linux/amd64": {
						Version: "test-1.0",
						URL:     srv.URL + "/aws.tar.gz",
						SHA256:  want,
						Archive: "tar.gz",
						Entry:   "dist/aws",
					},
				},
			},
		},
		CacheDir: cache,
		GOOS:     "linux",
		GOARCH:   "amd64",
	}
	_, err := r.Resolve("aws")
	if err == nil {
		t.Fatal("expected unsafe-path rejection, got nil")
	}
	if !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("err = %v, want unsafe path", err)
	}
}

func TestParseManifest_RejectsGarbage(t *testing.T) {
	if _, err := shell.ParseManifest([]byte("\t: : not yaml")); err == nil {
		t.Error("expected parse error for garbage")
	}
	// Valid YAML but no Tools map.
	b, _ := yaml.Marshal(map[string]string{"release_tag": "x"})
	if _, err := shell.ParseManifest(b); err == nil {
		t.Error("expected error for manifest with no tools map")
	}
}
