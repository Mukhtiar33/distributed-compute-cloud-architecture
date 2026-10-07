package auth

import (
	"testing"
	"time"
)

func TestTokenManager_IssueAndValidate(t *testing.T) {
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	token, err := tm.IssueToken("worker-1", "session", 5*time.Minute)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	claims, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.WorkerID != "worker-1" {
		t.Errorf("expected worker-1, got %s", claims.WorkerID)
	}
	if claims.TokenType != "session" {
		t.Errorf("expected session, got %s", claims.TokenType)
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	// Issue token that's already expired
	token, err := tm.IssueToken("worker-1", "session", -1*time.Minute)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	_, err = tm.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestTokenManager_InvalidToken(t *testing.T) {
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	_, err = tm.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}

func TestTokenManager_TamperedToken(t *testing.T) {
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	token, err := tm.IssueToken("worker-1", "session", 5*time.Minute)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	// Tamper with the token
	tampered := token[:len(token)-5] + "XXXXX"

	_, err = tm.ValidateToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}

func TestTokenManager_WrongWorkerID(t *testing.T) {
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	token, err := tm.IssueToken("worker-1", "session", 5*time.Minute)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	claims, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.WorkerID == "worker-2" {
		t.Error("worker ID should not match")
	}
}
