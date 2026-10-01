package session

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SessionManager handles session token acquisition and refresh.
type SessionManager struct {
	controlPlaneURL     string
	workerID           string
	enrollmentCredential string
	httpClient         *http.Client
	sessionToken       string
	expiresAt          time.Time
}

// NewSessionManager creates a new session manager.
func NewSessionManager(controlPlaneURL, workerID, enrollmentCredential string, caCertPEM []byte) (*SessionManager, error) {
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS12,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &SessionManager{
		controlPlaneURL:      controlPlaneURL,
		workerID:            workerID,
		enrollmentCredential: enrollmentCredential,
		httpClient:          &http.Client{Transport: transport, Timeout: 10 * time.Second},
	}, nil
}

// GetSessionToken returns a valid session token, refreshing if necessary.
func (sm *SessionManager) GetSessionToken() (string, error) {
	if sm.sessionToken != "" && time.Now().Before(sm.expiresAt.Add(-30*time.Second)) {
		return sm.sessionToken, nil
	}

	if err := sm.refresh(); err != nil {
		return "", err
	}
	return sm.sessionToken, nil
}

// refresh obtains a new session token from the Control Plane.
func (sm *SessionManager) refresh() error {
	reqBody, _ := json.Marshal(map[string]string{
		"worker_id":             sm.workerID,
		"enrollment_credential": sm.enrollmentCredential,
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
	sm.sessionToken = ""
	sm.expiresAt = time.Time{}
}
