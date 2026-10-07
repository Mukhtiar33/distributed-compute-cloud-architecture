package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
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
	keyPath    string
}

// NewTokenManager creates a TokenManager, loading existing key or generating new one.
func NewTokenManager(keyPath string) (*TokenManager, error) {
	tm := &TokenManager{keyPath: keyPath}

	// Try to load existing key
	if keyPath != "" {
		if _, err := os.Stat(keyPath); err == nil {
			keyPEM, err := os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read key file: %w", err)
			}

			block, _ := pem.Decode(keyPEM)
			if block == nil {
				return nil, fmt.Errorf("failed to decode key PEM")
			}

			privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}

			tm.privateKey = privKey
			tm.publicKey = &privKey.PublicKey
			return tm, nil
		}
	}

	// Generate new keypair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	tm.privateKey = privKey
	tm.publicKey = &privKey.PublicKey

	// Persist key if path provided
	if keyPath != "" {
		if err := os.MkdirAll(getDir(keyPath), 0700); err != nil {
			return nil, fmt.Errorf("failed to create key directory: %w", err)
		}

		keyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privKey),
		})

		if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
			return nil, fmt.Errorf("failed to write key file: %w", err)
		}
	}

	return tm, nil
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

func getDir(path string) string {
	lastSep := strings.LastIndex(path, "/")
	if lastSep == -1 {
		return "."
	}
	return path[:lastSep]
}
