package monitoring

import (
	"sync"
	"time"
)

// CacheMetrics tracks environment cache hit-rate (Document 15 §4).
type CacheMetrics struct {
	mu         sync.RWMutex
	hits       int
	misses     int
	lastHit    time.Time
	lastMiss   time.Time
}

// NewCacheMetrics creates a new cache metrics tracker.
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{}
}

// RecordHit records a cache hit.
func (cm *CacheMetrics) RecordHit() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.hits++
	cm.lastHit = time.Now()
}

// RecordMiss records a cache miss.
func (cm *CacheMetrics) RecordMiss() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.misses++
	cm.lastMiss = time.Now()
}

// GetHitRate returns the cache hit rate as a percentage.
func (cm *CacheMetrics) GetHitRate() float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	total := cm.hits + cm.misses
	if total == 0 {
		return 0
	}
	return float64(cm.hits) / float64(total) * 100
}

// GetStats returns cache statistics.
func (cm *CacheMetrics) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"hits":        cm.hits,
		"misses":      cm.misses,
		"hit_rate":    cm.GetHitRate(),
		"last_hit":    cm.lastHit,
		"last_miss":   cm.lastMiss,
	}
}

// Reset resets all metrics.
func (cm *CacheMetrics) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.hits = 0
	cm.misses = 0
	cm.lastHit = time.Time{}
	cm.lastMiss = time.Time{}
}
