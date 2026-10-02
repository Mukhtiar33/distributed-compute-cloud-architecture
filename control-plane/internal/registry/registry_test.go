package registry

import (
	"testing"
	"time"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry("/tmp/test-registry")

	env := &Environment{
		ID:        "python-3.11",
		Name:      "Python 3.11",
		Version:   "1.0.0",
		Hash:      "abc123",
		CreatedAt: time.Now(),
	}

	if err := r.Register(env); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	retrieved, ok := r.Get("python-3.11")
	if !ok {
		t.Fatal("expected to find environment")
	}
	if retrieved.Name != "Python 3.11" {
		t.Errorf("expected Python 3.11, got %s", retrieved.Name)
	}
}

func TestRegistry_GetNonexistent(t *testing.T) {
	r := NewRegistry("/tmp/test-registry")

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent environment")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry("/tmp/test-registry")

	r.Register(&Environment{ID: "env-1", Name: "Env 1", Hash: "hash1", CreatedAt: time.Now()})
	r.Register(&Environment{ID: "env-2", Name: "Env 2", Hash: "hash2", CreatedAt: time.Now()})

	envs := r.List()
	if len(envs) != 2 {
		t.Errorf("expected 2 environments, got %d", len(envs))
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := NewRegistry("/tmp/test-registry")

	env := &Environment{ID: "python-3.11", Name: "Python 3.11", Hash: "abc123", CreatedAt: time.Now()}
	r.Register(env)

	// Registering again should overwrite (update)
	env2 := &Environment{ID: "python-3.11", Name: "Python 3.11 v2", Hash: "def456", CreatedAt: time.Now()}
	r.Register(env2)

	retrieved, _ := r.Get("python-3.11")
	if retrieved.Name != "Python 3.11 v2" {
		t.Errorf("expected updated name, got %s", retrieved.Name)
	}
}

func TestRegistry_RegisterInvalid(t *testing.T) {
	r := NewRegistry("/tmp/test-registry")

	// Missing ID
	err := r.Register(&Environment{Name: "No ID", Hash: "abc"})
	if err == nil {
		t.Error("expected error for missing ID")
	}

	// Missing hash
	err = r.Register(&Environment{ID: "env-1", Name: "No Hash"})
	if err == nil {
		t.Error("expected error for missing hash")
	}
}

func TestHashContent(t *testing.T) {
	content1 := []byte("hello world")
	content2 := []byte("hello world")
	content3 := []byte("different content")

	hash1 := HashContent(content1)
	hash2 := HashContent(content2)
	hash3 := HashContent(content3)

	if hash1 != hash2 {
		t.Error("expected same hash for same content")
	}
	if hash1 == hash3 {
		t.Error("expected different hash for different content")
	}
	if len(hash1) != 64 { // SHA-256 hex string is 64 chars
		t.Errorf("expected 64 char hash, got %d", len(hash1))
	}
}

func TestHashContent_Deterministic(t *testing.T) {
	content := []byte("test content for hashing")

	hash1 := HashContent(content)
	hash2 := HashContent(content)

	if hash1 != hash2 {
		t.Error("hash should be deterministic")
	}
}
