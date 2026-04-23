package shell

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
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

	"gopkg.in/yaml.v3"
)

// embeddedManifest holds the in-tree YAML describing where to fetch each
// pinned tool binary and what its sha256 must be. Built into the coily
// binary at compile time. PRODUCTION USE NOTE: the placeholder SHAs in
// tools.yaml must be filled in by the .github/workflows/release-tools.yml
// workflow before any fetch will succeed. Until then, FetchingResolver
// errors at runtime when a real fetch is attempted.
//
//go:embed tools.yaml
var embeddedManifest []byte

// PlatformEntry pins one (tool, goos, goarch) to a fetch URL and a sha256
// the downloaded bytes must match. URL is where coily fetches from. The
// upstream_url field is metadata so a human (or the release workflow) can
// trace where the binary originally came from.
//
// When Archive is set, URL points at an archive (currently only
// "tar.gz" is supported) and Entry is the path within the archive to the
// executable coily should run. The entire archive is extracted into
// CacheDir/<sha256>/ so PyInstaller-style bundles (aws-cli v2) find
// their sibling files. When Archive is empty, the URL points at a
// standalone binary and the legacy single-file cache layout applies.
type PlatformEntry struct {
	Version     string `yaml:"version"`
	URL         string `yaml:"url"`
	UpstreamURL string `yaml:"upstream_url"`
	SHA256      string `yaml:"sha256"`
	Archive     string `yaml:"archive,omitempty"`
	Entry       string `yaml:"entry,omitempty"`
}

// ToolManifest is the structure of pkg/shell/tools.yaml.
type ToolManifest struct {
	ReleaseTag     string `yaml:"release_tag"`
	ReleaseURLBase string `yaml:"release_url_base"`
	// Tools maps tool name -> "goos/goarch" -> PlatformEntry.
	Tools map[string]map[string]PlatformEntry `yaml:"tools"`
}

// LoadEmbeddedManifest parses the in-tree tools.yaml that was embedded
// into the coily binary at build time.
func LoadEmbeddedManifest() (*ToolManifest, error) {
	return ParseManifest(embeddedManifest)
}

// ParseManifest parses a tools.yaml byte slice. Exported so tests can
// build a fixture manifest pointing at an httptest.Server.
func ParseManifest(b []byte) (*ToolManifest, error) {
	var m ToolManifest
	if err := yaml.Unmarshal(b, &m); err != nil {
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
	if e.Archive != "" && e.Archive != "tar.gz" {
		return PlatformEntry{}, fmt.Errorf("shell: tool %q on %s has unsupported archive format %q", tool, key, e.Archive)
	}
	if e.Archive != "" && e.Entry == "" {
		return PlatformEntry{}, fmt.Errorf("shell: tool %q on %s declares archive but no entry path", tool, key)
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
	// CacheDir/<sha256>/<tool> for single-binary entries and
	// CacheDir/<sha256>/<entry> for archive entries (the whole archive
	// is extracted under CacheDir/<sha256>/). Defaults to
	// ~/.cache/coily/bin when "".
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

// archiveSealMarker is the filename inside an extracted archive cache
// directory that signals a completed, verified extraction. Its presence
// short-circuits re-extraction on subsequent Resolve calls.
const archiveSealMarker = ".coily-archive-sha256"

// Resolve returns the absolute path to the cached binary for `bin`. On
// cache miss, fetches from the manifest URL, verifies sha256, writes to
// cache with mode 0700. Archive entries are extracted into
// CacheDir/<sha256>/ and the returned path points at <entry> within
// that tree. On checksum mismatch, the corrupt cache is removed and an
// error returned. Safe for concurrent calls within one process.
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

	if entry.Archive == "" {
		return f.resolveSingleFile(bin, entry, dir)
	}
	return f.resolveArchive(entry, cache, dir)
}

// resolveSingleFile is the legacy single-binary cache path. Kubectl and
// gh use this.
func (f *FetchingResolver) resolveSingleFile(bin string, entry PlatformEntry, dir string) (string, error) {
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

// resolveArchive downloads a tarball, verifies its sha256, extracts it
// into CacheDir/<sha256>/ atomically, and returns the path to the
// configured Entry. Unlike the single-file path, re-verification on
// cache hit is by presence of a sealed marker file rather than re-
// hashing the whole tree (cost prohibitive for aws-cli's ~60 MB
// bundle). The cache dir is mode 0700 so only the owning user can
// write, which matches the threat-model assumption already implicit in
// the single-file cache.
func (f *FetchingResolver) resolveArchive(entry PlatformEntry, cache, dir string) (string, error) {
	entryPath := filepath.Join(dir, entry.Entry)
	if markerOK(dir, entry.SHA256) {
		if _, err := os.Stat(entryPath); err == nil {
			return entryPath, nil
		}
	}
	// Either no marker, marker mismatch, or entry missing: drop any
	// partial extraction and re-fetch.
	_ = os.RemoveAll(dir)

	if err := os.MkdirAll(cache, 0o700); err != nil {
		return "", fmt.Errorf("shell: mkdir cache: %w", err)
	}
	staging, err := os.MkdirTemp(cache, ".coily-extract-*")
	if err != nil {
		return "", fmt.Errorf("shell: staging tempdir: %w", err)
	}
	// RemoveAll on error paths; rename makes it disappear on success.
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()

	if err := f.fetchExtractVerify(entry.URL, entry.SHA256, entry.Archive, staging); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, archiveSealMarker), []byte(entry.SHA256), 0o600); err != nil {
		return "", fmt.Errorf("shell: write seal marker: %w", err)
	}
	if err := os.Rename(staging, dir); err != nil {
		return "", fmt.Errorf("shell: commit extraction: %w", err)
	}
	committed = true
	if err := os.Chmod(entryPath, 0o700); err != nil {
		return "", fmt.Errorf("shell: chmod entry: %w", err)
	}
	return entryPath, nil
}

// markerOK returns true iff dir contains an archive seal marker with
// the expected sha256 string inside. The marker is written atomically
// after extraction completes, so its presence signals a clean, fully
// extracted bundle.
func markerOK(dir, wantSHA string) bool {
	b, err := os.ReadFile(filepath.Join(dir, archiveSealMarker))
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(b)), wantSHA)
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

// fetchExtractVerify downloads an archive from url, checksums the full
// byte stream against wantHex, and extracts into dest. The archive is
// buffered to a tempfile (not memory) so large bundles like aws-cli
// don't blow up RSS. Only tar.gz is accepted.
func (f *FetchingResolver) fetchExtractVerify(url, wantHex, archive, dest string) error {
	if archive != "tar.gz" {
		return fmt.Errorf("shell: unsupported archive format %q", archive)
	}
	tmp, err := f.downloadToTempVerifySHA(url, wantHex, dest)
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()
	return extractTarGz(tmp, dest)
}

// downloadToTempVerifySHA streams url into a fresh tempfile under dir,
// hashing as it goes. On hash mismatch the tempfile is closed, removed
// by the caller's defer, and an error is returned. On success the
// tempfile is returned rewound to offset 0.
func (f *FetchingResolver) downloadToTempVerifySHA(url, wantHex, dir string) (*os.File, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("shell: build fetch request: %w", err)
	}
	resp, err := f.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("shell: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shell: fetch %s: status %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(dir, ".coily-archive-*")
	if err != nil {
		return nil, fmt.Errorf("shell: tempfile: %w", err)
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("shell: read archive body: %w", err)
	}
	gotHex := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(gotHex, wantHex) {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("shell: sha256 mismatch for %s: got %s, want %s", url, gotHex, wantHex)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("shell: rewind archive: %w", err)
	}
	return tmp, nil
}

// extractTarGz reads a gzip-compressed tar stream from src and writes
// its entries under dest. Entries are validated before every write so
// no file lands outside dest.
func extractTarGz(src io.Reader, dest string) error {
	gz, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("shell: gzip open: %w", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("shell: tar next: %w", err)
		}
		if err := extractTarEntry(tr, hdr, dest); err != nil {
			return err
		}
	}
}

// extractTarEntry writes a single tar header's file/dir/symlink into
// dest. Paths are sanitized so no entry escapes dest (no absolute
// paths, no `..`, no symlinks that resolve outside the tree).
func extractTarEntry(tr *tar.Reader, hdr *tar.Header, dest string) error {
	target, err := safeTarTarget(hdr.Name, dest)
	if err != nil {
		return err
	}
	if target == "" {
		return nil
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, 0o700)
	case tar.TypeReg:
		return writeTarRegular(tr, hdr, target)
	case tar.TypeSymlink:
		return writeTarSymlink(hdr, target, dest)
	case tar.TypeLink:
		return writeTarHardLink(hdr, target, dest)
	default:
		// Skip device nodes, fifos, etc. - not expected in upstream
		// aws-cli / kubectl / gh bundles.
		return nil
	}
}

// safeTarTarget validates a tar entry name against dest. Returns the
// absolute target path or an error if the entry would escape dest.
// Returns ("", nil) for the no-op "." root entry.
func safeTarTarget(name, dest string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." {
		return "", nil
	}
	sep := string(filepath.Separator)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+sep) || strings.Contains(clean, sep+".."+sep) {
		return "", fmt.Errorf("shell: tar: unsafe path %q", name)
	}
	return filepath.Join(dest, clean), nil
}

// writeTarRegular copies a regular file entry's body to target,
// clamping mode bits to owner-only permissions.
func writeTarRegular(tr *tar.Reader, hdr *tar.Header, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("shell: tar: mkdir %s: %w", filepath.Dir(target), err)
	}
	// Clamp hdr.Mode (int64) into the 0o777 range before casting to
	// uint32 so gosec's overflow check is satisfied, then mask to
	// owner-only so extraction can't accidentally create
	// world-writable files.
	mode := os.FileMode(uint32(hdr.Mode&0o777)) & 0o700
	if mode == 0 {
		mode = 0o600
	}
	fp, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("shell: tar: open %s: %w", target, err)
	}
	// Guard against accidental decompression bombs. aws-cli is ~250 MB
	// uncompressed; 1 GB is plenty of headroom without letting a
	// pathological archive eat the disk.
	const maxFileSize = 1 << 30
	if _, err := io.CopyN(fp, tr, maxFileSize); err != nil && !errors.Is(err, io.EOF) {
		_ = fp.Close()
		return fmt.Errorf("shell: tar: write %s: %w", target, err)
	}
	if err := fp.Close(); err != nil {
		return fmt.Errorf("shell: tar: close %s: %w", target, err)
	}
	return nil
}

// writeTarSymlink creates a symlink entry after verifying its target
// does not escape dest.
func writeTarSymlink(hdr *tar.Header, target, dest string) error {
	if filepath.IsAbs(hdr.Linkname) {
		return fmt.Errorf("shell: tar: absolute symlink %q -> %q", hdr.Name, hdr.Linkname)
	}
	// Resolve lexically inside dest and reject anything that leaves
	// the tree. filepath.Join below is safe because we've just
	// rejected the escape case. gosec G305 flags Join-inside-extract
	// without this lexical check; the check is the mitigation.
	resolved := filepath.Clean(filepath.Join(filepath.Dir(target), hdr.Linkname)) //nolint:gosec // escape-check below
	cleanDest := filepath.Clean(dest)
	sep := string(filepath.Separator)
	if resolved != cleanDest && !strings.HasPrefix(resolved, cleanDest+sep) {
		return fmt.Errorf("shell: tar: symlink escapes dest: %q -> %q", hdr.Name, hdr.Linkname)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("shell: tar: mkdir %s: %w", filepath.Dir(target), err)
	}
	if err := os.Symlink(hdr.Linkname, target); err != nil {
		return fmt.Errorf("shell: tar: symlink %s -> %s: %w", target, hdr.Linkname, err)
	}
	return nil
}

// writeTarHardLink creates a hard link after verifying the source
// path stays inside dest.
func writeTarHardLink(hdr *tar.Header, target, dest string) error {
	// See writeTarSymlink for gosec G305 rationale.
	source := filepath.Join(dest, filepath.Clean(hdr.Linkname)) //nolint:gosec // escape-check below
	cleanDest := filepath.Clean(dest)
	sep := string(filepath.Separator)
	if source != cleanDest && !strings.HasPrefix(source, cleanDest+sep) {
		return fmt.Errorf("shell: tar: hard link escapes dest: %q -> %q", hdr.Name, hdr.Linkname)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("shell: tar: mkdir %s: %w", filepath.Dir(target), err)
	}
	if err := os.Link(source, target); err != nil {
		return fmt.Errorf("shell: tar: hard link %s -> %s: %w", target, source, err)
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
