package session

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SessionManager handles session token acquisition and refresh.
// Session tokens are obtained using mTLS identity, not enrollment credentials.
type SessionManager struct {
	controlPlaneURL string
	workerID       string
	httpClient     *http.Client
	sessionToken   string
	expiresAt      time.Time
	mu             sync.Mutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager(controlPlaneURL, workerID string, caCertPEM []byte, clientCert tls.Certificate) (*SessionManager, error) {
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &SessionManager{
		controlPlaneURL: controlPlaneURL,
		workerID:       workerID,
		httpClient:     &http.Client{Transport: transport, Timeout: 10 * time.Second},
	}, nil
}

// GetSessionToken returns a valid session token, refreshing if necessary.
func (sm *SessionManager) GetSessionToken() (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.sessionToken != "" && time.Now().Before(sm.expiresAt.Add(-30*time.Second)) {
		return sm.sessionToken, nil
	}

	if err := sm.refresh(); err != nil {
		return "", err
	}
	return sm.sessionToken, nil
}

// refresh obtains a new session token from the Control Plane using mTLS identity.
func (sm *SessionManager) refresh() error {
	// Session establishment uses mTLS peer certificate identity
	// No enrollment credential is sent over the wire
	reqBody, _ := json.Marshal(map[string]string{
		"worker_id": sm.workerID,
	})

	resp, err := sm.httpClient.Post(
		sm.controlPlaneURL+"/session",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("session request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("session request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		SessionToken string `json:"session_token"`
		ExpiresAt    int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode session response: %w", err)
	}

	sm.sessionToken = result.SessionToken
	sm.expiresAt = time.Unix(result.ExpiresAt, 0)
	return nil
}

// ForceRefresh forces a token refresh on the next GetSessionToken call.
func (sm *SessionManager) ForceRefresh() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessionToken = ""
	sm.expiresAt = time.Time{}
}
