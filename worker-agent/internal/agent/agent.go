package agent

import (
	"log"
	"sync"

	"github.com/distributedcompute/cloud/worker-agent/internal/config"
)

// Agent represents the Worker Agent running on a provider's machine.
type Agent struct {
	cfg    config.Config
	mu     sync.Mutex
	running bool
}

// New creates a new Worker Agent instance.
func New(cfg config.Config) *Agent {
	return &Agent{cfg: cfg}
}

// Start initializes the agent and begins its main loop.
// In Phase 0, this is a minimal stub that logs startup.
// Phase 1 will add enrollment, heartbeat, and mTLS.
func (a *Agent) Start() error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()

	log.Printf("Worker Agent started, will connect to %s", a.cfg.ControlPlaneURL)
	return nil
}

// Stop gracefully shuts down the agent.
func (a *Agent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running = false
	log.Println("Worker Agent stopped")
}

// IsRunning reports whether the agent is currently active.
func (a *Agent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// Daemonize runs the agent as a background service.
// On Linux/macOS, this forks the process into the background.
// On Windows, this is a no-op (use Windows Services for production).
func Daemonize() {
	daemonize()
}
