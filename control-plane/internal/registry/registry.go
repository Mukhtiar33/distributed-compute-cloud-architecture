package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Environment represents a registered execution environment.
type Environment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Hash        string            `json:"hash"`         // SHA-256 hash of contents
	ContentPath string            `json:"-"`            // Path to environment content on disk
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Registry manages the environment catalog.
type Registry struct {
	mu          sync.RWMutex
	environments map[string]*Environment // key: environment ID
	storePath   string
}

// NewRegistry creates a new environment registry.
func NewRegistry(storePath string) *Registry {
	return &Registry{
		environments: make(map[string]*Environment),
		storePath:   storePath,
	}
}

// Register adds a new environment to the registry.
func (r *Registry) Register(env *Environment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if env.ID == "" {
		return fmt.Errorf("environment ID is required")
	}
	if env.Hash == "" {
		return fmt.Errorf("environment hash is required")
	}

	r.environments[env.ID] = env
	return nil
}

// Get retrieves an environment by ID.
func (r *Registry) Get(id string) (*Environment, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	env, ok := r.environments[id]
	return env, ok
}

// List returns all registered environments.
func (r *Registry) List() []*Environment {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Environment, 0, len(r.environments))
	for _, env := range r.environments {
		result = append(result, env)
	}
	return result
}

// VerifyHash checks if the content at the given path matches the expected hash.
func (r *Registry) VerifyHash(contentPath, expectedHash string) (bool, error) {
	hash, err := computeFileHash(contentPath)
	if err != nil {
		return false, err
	}
	return hash == expectedHash, nil
}

// ComputeHash computes the SHA-256 hash of a file's contents.
func ComputeHash(contentPath string) (string, error) {
	return computeFileHash(contentPath)
}

func computeFileHash(path string) (string, error) {
	// In a real implementation, this would read the file and compute the hash
	// For Phase 5, we use a simple approach
	return "", fmt.Errorf("not implemented")
}

// GetStorePath returns the base path for environment storage.
func (r *Registry) GetStorePath() string {
	return r.storePath
}

// HashContent computes a SHA-256 hash of the given content.
func HashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
