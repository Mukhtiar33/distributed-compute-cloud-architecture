package session

import (
	"testing"
	"time"
)

func TestSessionManager_ForceRefresh(t *testing.T) {
	sm := &SessionManager{
		workerID:            "worker-test",
		enrollmentCredential: "test-cred",
		sessionToken:        "old-token",
		expiresAt:           time.Now().Add(5 * time.Minute),
	}

	sm.ForceRefresh()

	if sm.sessionToken != "" {
		t.Error("session token should be empty after ForceRefresh")
	}
	if !sm.expiresAt.IsZero() {
		t.Error("expiresAt should be zero after ForceRefresh")
	}
}

func TestSessionManager_GetSessionTokenCached(t *testing.T) {
	sm := &SessionManager{
		workerID:            "worker-test",
		enrollmentCredential: "test-cred",
		sessionToken:        "cached-token",
		expiresAt:           time.Now().Add(5 * time.Minute),
	}

	// Should return cached token without refreshing
	token, err := sm.GetSessionToken()
	if err != nil {
		t.Fatalf("GetSessionToken failed: %v", err)
	}
	if token != "cached-token" {
		t.Errorf("expected cached-token, got %s", token)
	}
}
