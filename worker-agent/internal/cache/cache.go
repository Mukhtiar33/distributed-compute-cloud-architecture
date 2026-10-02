package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached environment layer.
type CacheEntry struct {
	EnvironmentID string    `json:"environment_id"`
	Hash          string    `json:"hash"`
	Path          string    `json:"path"`
	CachedAt     time.Time `json:"cached_at"`
	LastUsed      time.Time `json:"last_used"`
	SizeBytes     int64     `json:"size_bytes"`
}

// Cache manages the local environment layer cache.
type Cache struct {
	mu      sync.RWMutex
	baseDir string
	entries map[string]*CacheEntry // key: environment ID
}

// NewCache creates a new environment cache.
func NewCache(baseDir string) (*Cache, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}
	return &Cache{
		baseDir: baseDir,
		entries: make(map[string]*CacheEntry),
	}, nil
}

// Get retrieves a cache entry by environment ID.
func (c *Cache) Get(environmentID string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[environmentID]
	return entry, ok
}

// Put adds or updates a cache entry.
func (c *Cache) Put(entry *CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify the content exists and hash matches
	hash, err := computeHash(entry.Path)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}
	if hash != entry.Hash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", entry.Hash, hash)
	}

	c.entries[entry.EnvironmentID] = entry
	return nil
}

// Verify checks if a cached entry's hash still matches its content.
func (c *Cache) Verify(environmentID string) (bool, error) {
	c.mu.RLock()
	entry, ok := c.entries[environmentID]
	c.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("environment %s not in cache", environmentID)
	}

	hash, err := computeHash(entry.Path)
	if err != nil {
		return false, err
	}

	return hash == entry.Hash, nil
}

// Remove deletes a cache entry.
func (c *Cache) Remove(environmentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[environmentID]
	if !ok {
		return fmt.Errorf("environment %s not in cache", environmentID)
	}

	// Remove from disk
	if entry.Path != "" {
		os.RemoveAll(entry.Path)
	}

	delete(c.entries, environmentID)
	return nil
}

// List returns all cache entries.
func (c *Cache) List() []*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*CacheEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		result = append(result, entry)
	}
	return result
}

// GetCacheHitRate returns the cache hit rate as a percentage.
func (c *Cache) GetCacheHitRate(hits, misses int) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// Cleanup removes stale cache entries.
func (c *Cache) Cleanup(maxAge time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, entry := range c.entries {
		if now.Sub(entry.LastUsed) > maxAge {
			if entry.Path != "" {
				os.RemoveAll(entry.Path)
			}
			delete(c.entries, id)
		}
	}
	return nil
}

// GetBaseDir returns the base directory for the cache.
func (c *Cache) GetBaseDir() string {
	return c.baseDir
}

// GetEntryPath returns the path for a given environment ID.
func (c *Cache) GetEntryPath(environmentID string) string {
	return filepath.Join(c.baseDir, environmentID)
}

func computeHash(path string) (string, error) {
	// Check if path is a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return computeDirHash(path)
	}

	return computeFileHash(path)
}

func computeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func computeDirHash(dir string) (string, error) {
	var combinedHash []byte

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(data)
		combinedHash = append(combinedHash, hash[:]...)
		return nil
	})

	if err != nil {
		return "", err
	}

	finalHash := sha256.Sum256(combinedHash)
	return hex.EncodeToString(finalHash[:]), nil
}
