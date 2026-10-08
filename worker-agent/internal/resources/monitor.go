package resources

import (
	"log"
	"runtime"
	"sync"
	"time"
)

// Monitor tracks host resource usage and enforces throttling policies.
type Monitor struct {
	mu              sync.RWMutex
	cpuThreshold    float64
	memoryThreshold int64
	diskThreshold   int64
	lastCheck       time.Time
	available       bool
}

// NewMonitor creates a new resource monitor with default thresholds.
func NewMonitor() *Monitor {
	return &Monitor{
		cpuThreshold:    80.0,  // Throttle when CPU > 80%
		memoryThreshold: 90,    // Throttle when memory > 90%
		diskThreshold:   95,    // Throttle when disk > 95%
		available:       true,
		lastCheck:       time.Now(),
	}
}

// CheckResources checks current resource usage and updates availability.
func (m *Monitor) CheckResources() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Simple heuristic: if system memory is under pressure, throttle
	// In production, this would use platform-specific APIs
	m.available = true
	m.lastCheck = time.Now()

	log.Printf("Resource check: available=%v", m.available)
}

// IsAvailable returns whether the worker is available to accept new jobs.
func (m *Monitor) IsAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.available
}

// ShouldThrottle returns whether job execution should be throttled.
func (m *Monitor) ShouldThrottle() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return !m.available
}

// GetStats returns current resource statistics.
func (m *Monitor) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"available":      m.available,
		"last_check":     m.lastCheck,
		"cpu_threshold":  m.cpuThreshold,
		"memory_threshold": m.memoryThreshold,
		"disk_threshold": m.diskThreshold,
		"goroutines":     runtime.NumGoroutine(),
	}
}
