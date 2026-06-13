package wiki

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// Cache is a tiny on-disk blob cache keyed by an arbitrary string, with a TTL
// per entry. It is safe for the simple single-process use the CLI makes of it.
type Cache struct {
	dir     string
	enabled bool
}

// NewCache returns a cache rooted under dir. If dir is empty or enabled is
// false, all operations are no-ops (cache miss on every Get).
func NewCache(dir string, enabled bool) *Cache {
	return &Cache{dir: dir, enabled: enabled && dir != ""}
}

func (c *Cache) pathFor(key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".cache")
}

// Get returns cached bytes for key if present and younger than ttl.
func (c *Cache) Get(key string, ttl time.Duration) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}
	p := c.pathFor(key)
	info, err := os.Stat(p)
	if err != nil {
		return nil, false
	}
	if ttl > 0 && time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Put stores data under key.
func (c *Cache) Put(key string, data []byte) {
	if !c.enabled {
		return
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return
	}
	p := c.pathFor(key)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err == nil {
		_ = os.Rename(tmp, p)
	}
}

// Clear removes every cached entry. It returns the number of files removed.
func (c *Cache) Clear() (int, error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".cache" {
			if os.Remove(filepath.Join(c.dir, e.Name())) == nil {
				n++
			}
		}
	}
	return n, nil
}

// Info returns the number of cache entries and their total size in bytes.
func (c *Cache) Info() (count int, bytes int64) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".cache" {
			continue
		}
		if info, err := e.Info(); err == nil {
			count++
			bytes += info.Size()
		}
	}
	return count, bytes
}

// Dir returns the cache directory.
func (c *Cache) Dir() string { return c.dir }
