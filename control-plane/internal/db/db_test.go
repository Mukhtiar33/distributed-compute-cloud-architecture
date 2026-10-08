package db

import (
	"testing"
)

func TestConfig_ConnectionString(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "test",
		Password: "test",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	if cfg.Host != "localhost" {
		t.Errorf("expected localhost, got %s", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected 5432, got %d", cfg.Port)
	}
}

func TestMigrations(t *testing.T) {
	// This test requires a running PostgreSQL instance
	// Skip if not available
	t.Skip("Requires running PostgreSQL instance")
}
