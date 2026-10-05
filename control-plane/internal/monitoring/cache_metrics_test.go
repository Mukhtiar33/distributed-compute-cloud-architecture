package monitoring

import (
	"testing"
	"time"
)

func TestCacheMetrics_RecordHit(t *testing.T) {
	cm := NewCacheMetrics()
	cm.RecordHit()

	stats := cm.GetStats()
	if stats["hits"].(int) != 1 {
		t.Errorf("expected 1 hit, got %d", stats["hits"])
	}
}

func TestCacheMetrics_RecordMiss(t *testing.T) {
	cm := NewCacheMetrics()
	cm.RecordMiss()

	stats := cm.GetStats()
	if stats["misses"].(int) != 1 {
		t.Errorf("expected 1 miss, got %d", stats["misses"])
	}
}

func TestCacheMetrics_GetHitRate(t *testing.T) {
	cm := NewCacheMetrics()

	// No data
	if cm.GetHitRate() != 0 {
		t.Errorf("expected 0%% hit rate, got %f", cm.GetHitRate())
	}

	// 80 hits, 20 misses = 80%
	for i := 0; i < 80; i++ {
		cm.RecordHit()
	}
	for i := 0; i < 20; i++ {
		cm.RecordMiss()
	}

	rate := cm.GetHitRate()
	if rate != 80.0 {
		t.Errorf("expected 80%% hit rate, got %f", rate)
	}
}

func TestCacheMetrics_GetStats(t *testing.T) {
	cm := NewCacheMetrics()
	cm.RecordHit()
	cm.RecordMiss()

	stats := cm.GetStats()
	if stats["hits"].(int) != 1 {
		t.Errorf("expected 1 hit, got %d", stats["hits"])
	}
	if stats["misses"].(int) != 1 {
		t.Errorf("expected 1 miss, got %d", stats["misses"])
	}
	if stats["hit_rate"].(float64) != 50.0 {
		t.Errorf("expected 50%% hit rate, got %f", stats["hit_rate"])
	}
}

func TestCacheMetrics_Reset(t *testing.T) {
	cm := NewCacheMetrics()
	cm.RecordHit()
	cm.RecordMiss()

	cm.Reset()

	stats := cm.GetStats()
	if stats["hits"].(int) != 0 {
		t.Errorf("expected 0 hits after reset, got %d", stats["hits"])
	}
	if stats["misses"].(int) != 0 {
		t.Errorf("expected 0 misses after reset, got %d", stats["misses"])
	}
}

func TestCacheMetrics_AllHits(t *testing.T) {
	cm := NewCacheMetrics()
	for i := 0; i < 100; i++ {
		cm.RecordHit()
	}

	if cm.GetHitRate() != 100.0 {
		t.Errorf("expected 100%% hit rate, got %f", cm.GetHitRate())
	}
}

func TestCacheMetrics_AllMisses(t *testing.T) {
	cm := NewCacheMetrics()
	for i := 0; i < 100; i++ {
		cm.RecordMiss()
	}

	if cm.GetHitRate() != 0.0 {
		t.Errorf("expected 0%% hit rate, got %f", cm.GetHitRate())
	}
}

func TestCacheMetrics_ConcurrentAccess(t *testing.T) {
	cm := NewCacheMetrics()

	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			cm.RecordHit()
			done <- true
		}()
		go func() {
			cm.RecordMiss()
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	stats := cm.GetStats()
	if stats["hits"].(int) != 10 {
		t.Errorf("expected 10 hits, got %d", stats["hits"])
	}
	if stats["misses"].(int) != 10 {
		t.Errorf("expected 10 misses, got %d", stats["misses"])
	}
}

func TestCacheMetrics_LastHit(t *testing.T) {
	cm := NewCacheMetrics()
	cm.RecordHit()

	stats := cm.GetStats()
	lastHit := stats["last_hit"].(time.Time)
	if lastHit.IsZero() {
		t.Error("expected last_hit to be set")
	}
}

func TestCacheMetrics_LastMiss(t *testing.T) {
	cm := NewCacheMetrics()
	cm.RecordMiss()

	stats := cm.GetStats()
	lastMiss := stats["last_miss"].(time.Time)
	if lastMiss.IsZero() {
		t.Error("expected last_miss to be set")
	}
}
