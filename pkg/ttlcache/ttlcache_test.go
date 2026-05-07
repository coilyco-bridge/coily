package ttlcache_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coilysiren/coily/pkg/ttlcache"
)

func TestSetGet_RoundTrip(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	if err := c.Set("k1", []byte("v1"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok := c.Get("k1")
	if !ok {
		t.Fatal("Get: miss after Set")
	}
	if string(got) != "v1" {
		t.Errorf("Get = %q, want %q", got, "v1")
	}
}

func TestGet_MissOnEmpty(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	if _, ok := c.Get("nonexistent"); ok {
		t.Error("Get: hit on empty cache")
	}
}

func TestGet_MissOnExpired(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	// 1 nanosecond TTL guarantees an expired entry by the time Get runs.
	if err := c.Set("k", []byte("v"), time.Nanosecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Error("Get: hit on expired entry")
	}
}

func TestGet_MissOnCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	c := ttlcache.New(dir)
	if err := c.Set("k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// Walk the dir to find the single entry, overwrite with garbage.
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("Glob: matches=%v err=%v", matches, err)
	}
	if err := os.WriteFile(matches[0], []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, ok := c.Get("k"); ok {
		t.Error("Get: hit on corrupted entry; should miss + let caller refetch")
	}
}

func TestGetOrSet_FetchOnMiss(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	calls := 0
	fetch := func() ([]byte, error) {
		calls++
		return []byte("fresh"), nil
	}
	v, err := c.GetOrSet("k", time.Minute, fetch)
	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if string(v) != "fresh" {
		t.Errorf("first GetOrSet = %q, want %q", v, "fresh")
	}
	v, err = c.GetOrSet("k", time.Minute, fetch)
	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if string(v) != "fresh" {
		t.Errorf("second GetOrSet = %q, want %q", v, "fresh")
	}
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1 (second call should hit cache)", calls)
	}
}

func TestGetOrSet_FetchErrorPropagates(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	wantErr := errors.New("upstream failure")
	_, err := c.GetOrSet("k", time.Minute, func() ([]byte, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("GetOrSet error = %v, want errors.Is(_, wantErr)", err)
	}
	// Cache must not store on fetch error.
	if _, ok := c.Get("k"); ok {
		t.Error("Get: hit after a failed fetch; should not store")
	}
}

func TestInvalidate_RemovesEntry(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	if err := c.Set("k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Invalidate("k"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	if _, ok := c.Get("k"); ok {
		t.Error("Get: hit after Invalidate")
	}
}

func TestInvalidate_IsIdempotent(t *testing.T) {
	c := ttlcache.New(t.TempDir())
	if err := c.Invalidate("never-set"); err != nil {
		t.Errorf("Invalidate on missing key: %v", err)
	}
}
