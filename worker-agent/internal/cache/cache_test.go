package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := NewCache(tmpDir)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test-env.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	hash, _ := computeFileHash(testFile)

	entry := &CacheEntry{
		EnvironmentID: "env-1",
		Hash:          hash,
		Path:          testFile,
		CachedAt:      time.Now(),
		LastUsed:      time.Now(),
	}

	if err := c.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, ok := c.Get("env-1")
	if !ok {
		t.Fatal("expected to find cache entry")
	}
	if retrieved.Hash != hash {
		t.Errorf("expected hash %s, got %s", hash, retrieved.Hash)
	}
}

func TestCache_GetNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent entry")
	}
}

func TestCache_Verify(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	testFile := filepath.Join(tmpDir, "test-env.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	hash, _ := computeFileHash(testFile)

	entry := &CacheEntry{
		EnvironmentID: "env-1",
		Hash:          hash,
		Path:          testFile,
		CachedAt:      time.Now(),
		LastUsed:      time.Now(),
	}
	c.Put(entry)

	valid, err := c.Verify("env-1")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Error("expected hash to be valid")
	}
}

func TestCache_VerifyTampered(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	testFile := filepath.Join(tmpDir, "test-env.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	hash, _ := computeFileHash(testFile)

	entry := &CacheEntry{
		EnvironmentID: "env-1",
		Hash:          hash,
		Path:          testFile,
		CachedAt:      time.Now(),
		LastUsed:      time.Now(),
	}
	c.Put(entry)

	// Tamper with the file
	os.WriteFile(testFile, []byte("tampered content"), 0644)

	valid, err := c.Verify("env-1")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Error("expected hash to be invalid after tampering")
	}
}

func TestCache_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	testFile := filepath.Join(tmpDir, "test-env.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	hash, _ := computeFileHash(testFile)

	entry := &CacheEntry{
		EnvironmentID: "env-1",
		Hash:          hash,
		Path:          testFile,
		CachedAt:      time.Now(),
		LastUsed:      time.Now(),
	}
	c.Put(entry)

	if err := c.Remove("env-1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, ok := c.Get("env-1")
	if ok {
		t.Error("expected entry to be removed")
	}
}

func TestCache_List(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	// Add two entries
	for _, id := range []string{"env-1", "env-2"} {
		testFile := filepath.Join(tmpDir, id+".txt")
		os.WriteFile(testFile, []byte("content"), 0644)
		hash, _ := computeFileHash(testFile)

		c.Put(&CacheEntry{
			EnvironmentID: id,
			Hash:          hash,
			Path:          testFile,
			CachedAt:      time.Now(),
			LastUsed:      time.Now(),
		})
	}

	entries := c.List()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestCache_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	testFile := filepath.Join(tmpDir, "test-env.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	hash, _ := computeFileHash(testFile)

	entry := &CacheEntry{
		EnvironmentID: "env-1",
		Hash:          hash,
		Path:          testFile,
		CachedAt:      time.Now(),
		LastUsed:      time.Now().Add(-2 * time.Hour), // Old entry
	}
	c.Put(entry)

	// Cleanup entries older than 1 hour
	if err := c.Cleanup(1 * time.Hour); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	_, ok := c.Get("env-1")
	if ok {
		t.Error("expected old entry to be cleaned up")
	}
}

func TestCache_GetCacheHitRate(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	rate := c.GetCacheHitRate(80, 20)
	if rate != 80.0 {
		t.Errorf("expected 80%% hit rate, got %f", rate)
	}

	rate = c.GetCacheHitRate(0, 0)
	if rate != 0 {
		t.Errorf("expected 0%% hit rate for no requests, got %f", rate)
	}
}

func TestCache_PutHashMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	testFile := filepath.Join(tmpDir, "test-env.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	entry := &CacheEntry{
		EnvironmentID: "env-1",
		Hash:          "wrong-hash",
		Path:          testFile,
		CachedAt:      time.Now(),
		LastUsed:      time.Now(),
	}

	err := c.Put(entry)
	if err == nil {
		t.Error("expected error for hash mismatch")
	}
}

func TestCache_GetBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	if c.GetBaseDir() != tmpDir {
		t.Errorf("expected base dir %s, got %s", tmpDir, c.GetBaseDir())
	}
}

func TestCache_GetEntryPath(t *testing.T) {
	tmpDir := t.TempDir()
	c, _ := NewCache(tmpDir)

	path := c.GetEntryPath("env-1")
	expected := filepath.Join(tmpDir, "env-1")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
