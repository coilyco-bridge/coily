// Package ttlcache is a small on-disk key/value cache with per-entry TTLs.
// Built for coily after the lockdown inversion: every aws / kubectl / gh
// call now round-trips through coily, and several of those (sts
// get-caller-identity, gh auth status, git rev-parse --show-toplevel) are
// read-mostly with results that are stable for minutes at a time.
//
// Why a CLI deserves a cache: each coily invocation is a fresh process, so
// in-memory memoization buys nothing. An on-disk cache under
// ~/.coily/cache/ persists across invocations and keeps repeat-call latency
// low without giving up the "every invocation is auditable" property -
// audit-log writes happen at the verb layer, independent of whether the
// underlying read came from cache.
//
// Failure mode is "fall through to the underlying call." A corrupted
// entry, a permissions error, or a clock skew that makes everything look
// stale all degrade to "fetch fresh and overwrite." No cached entry is
// ever load-bearing for correctness; a cache miss is always recoverable.
package ttlcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Cache is a directory of TTL'd JSON entries on disk. One file per key,
// named by sha256(key) so disk layout stays bounded and predictable. The
// zero value is unusable; build with New.
type Cache struct {
	Dir string
}

// New returns a Cache rooted at dir. The directory is created lazily on
// first Set, with mode 0o700 so other local users cannot read cached
// values. Callers typically pass filepath.Join(home, ".coily", "cache").
func New(dir string) *Cache {
	return &Cache{Dir: dir}
}

// entry is the on-disk shape. Keeping the TTL in the file (rather than in
// the cache instance) means a single cache directory can hold entries
// with different freshness windows.
type entry struct {
	Value      []byte    `json:"value"`
	StoredAt   time.Time `json:"stored_at"`
	TTLSeconds int       `json:"ttl_seconds"`
}

func (c *Cache) keyToPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.Dir, hex.EncodeToString(h[:])+".json")
}

// Get returns the cached value for key, or (nil, false) if there is no
// fresh entry. Any read / unmarshal / TTL failure is reported as a miss
// rather than an error - the contract is "you got a value or you didn't,"
// and callers should fall through to the underlying fetch on miss.
func (c *Cache) Get(key string) ([]byte, bool) {
	data, err := os.ReadFile(c.keyToPath(key))
	if err != nil {
		return nil, false
	}
	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}
	if time.Since(e.StoredAt) > time.Duration(e.TTLSeconds)*time.Second {
		return nil, false
	}
	return e.Value, true
}

// Set writes value under key with the given TTL. Returns the underlying
// filesystem error if the write fails; callers can ignore it (the next
// Get will simply miss).
func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(entry{
		Value:      value,
		StoredAt:   time.Now(),
		TTLSeconds: int(ttl.Seconds()),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(c.keyToPath(key), data, 0o600)
}

// GetOrSet returns the cached value for key, or calls fetch() and stores
// the result if there is no fresh entry. fetch is called at most once.
//
// A fetch error is returned as-is; the cache is not updated. A Set error
// after a successful fetch is swallowed (the value is returned anyway) -
// the cache is a perf hint, not a correctness gate.
func (c *Cache) GetOrSet(key string, ttl time.Duration, fetch func() ([]byte, error)) ([]byte, error) {
	if v, ok := c.Get(key); ok {
		return v, nil
	}
	v, err := fetch()
	if err != nil {
		return nil, err
	}
	_ = c.Set(key, v, ttl) // perf hint, not load-bearing
	return v, nil
}

// Invalidate removes the entry for key. Returns nil if the entry did not
// exist (idempotent), the filesystem error otherwise. Used by callers
// that observed a stale read and want to force a refetch on the next
// call.
func (c *Cache) Invalidate(key string) error {
	err := os.Remove(c.keyToPath(key))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
