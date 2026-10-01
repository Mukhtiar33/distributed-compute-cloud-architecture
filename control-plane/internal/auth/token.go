package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// SessionTokenLifetime is how long a session token is valid.
	SessionTokenLifetime = 5 * time.Minute
	// EnrollmentTokenLifetime is how long an enrollment credential is valid.
	EnrollmentTokenLifetime = 24 * time.Hour
)

// TokenClaims represents the payload of a signed token.
type TokenClaims struct {
	WorkerID  string `json:"worker_id"`
	TokenType string `json:"token_type"` // "enrollment" or "session"
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
}

// TokenManager issues and validates signed tokens.
type TokenManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	mu         sync.RWMutex
}

// NewTokenManager creates a TokenManager with a newly generated RSA keypair.
func NewTokenManager() (*TokenManager, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	return &TokenManager{
		privateKey: privKey,
		publicKey:  &privKey.PublicKey,
	}, nil
}

// NewTokenManagerWithKey creates a TokenManager with an existing private key.
func NewTokenManagerWithKey(privateKey *rsa.PrivateKey) *TokenManager {
	return &TokenManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}
}

// IssueToken creates a signed token for the given worker.
func (tm *TokenManager) IssueToken(workerID, tokenType string, lifetime time.Duration) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		WorkerID:  workerID,
		TokenType: tokenType,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(lifetime).Unix(),
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	hash := sha256.Sum256(claimsJSON)
	signature, err := rsa.SignPKCS1v15(rand.Reader, tm.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	tokenData := base64.RawURLEncoding.EncodeToString(claimsJSON) + "." +
		base64.RawURLEncoding.EncodeToString(signature)
	return tokenData, nil
}

// ValidateToken verifies a token's signature and expiry.
func (tm *TokenManager) ValidateToken(token string) (*TokenClaims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	hash := sha256.Sum256(claimsJSON)
	if err := rsa.VerifyPKCS1v15(tm.publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return nil, fmt.Errorf("invalid token signature: %w", err)
	}

	var claims TokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// GetPublicKey returns the public key for external verification.
func (tm *TokenManager) GetPublicKey() *rsa.PublicKey {
	return tm.publicKey
}
