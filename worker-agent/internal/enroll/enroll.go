package enroll

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// EnrollClient handles worker enrollment with the Control Plane.
type EnrollClient struct {
	controlPlaneURL string
	httpClient      *http.Client
}

// NewEnrollClient creates a new enrollment client.
func NewEnrollClient(controlPlaneURL string, caCertPEM []byte) (*EnrollClient, error) {
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS12,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &EnrollClient{
		controlPlaneURL: controlPlaneURL,
		httpClient:      &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}, nil
}

// EnrollResult contains the artifacts from a successful enrollment.
type EnrollResult struct {
	WorkerID             string
	EnrollmentCredential string
	ClientCertificate    []byte
	ClientKey            []byte
}

// Enroll performs the full enrollment flow: generate keypair, create CSR, submit to Control Plane.
func (c *EnrollClient) Enroll(workerID string) (*EnrollResult, error) {
	// Generate RSA keypair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Create CSR
	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   workerID,
			Organization: []string{"Distributed Compute Cloud"},
		},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	// Submit enrollment request
	reqBody, _ := json.Marshal(map[string]string{
		"worker_id": workerID,
		"csr":       string(csrPEM),
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/enroll",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("enrollment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enrollment failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		EnrollmentCredential string `json:"enrollment_credential"`
		ClientCertificate    string `json:"client_certificate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode enrollment response: %w", err)
	}

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	return &EnrollResult{
		WorkerID:             workerID,
		EnrollmentCredential: result.EnrollmentCredential,
		ClientCertificate:    []byte(result.ClientCertificate),
		ClientKey:            keyPEM,
	}, nil
}

// SaveCredentials persists enrollment credentials to disk for reuse on restart.
func (c *EnrollClient) SaveCredentials(result *EnrollResult, dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials dir: %w", err)
	}

	files := map[string][]byte{
		"worker_id":              []byte(result.WorkerID),
		"enrollment_credential":  []byte(result.EnrollmentCredential),
		"client_certificate.pem": result.ClientCertificate,
		"client_key.pem":         result.ClientKey,
	}

	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}

// LoadCredentials loads previously saved enrollment credentials from disk.
func (c *EnrollClient) LoadCredentials(dir string) (*EnrollResult, error) {
	files := map[string]string{
		"worker_id":              "worker_id",
		"enrollment_credential":  "enrollment_credential",
		"client_certificate.pem": "client_certificate",
		"client_key.pem":         "client_key",
	}

	result := &EnrollResult{}
	for filename, field := range files {
		path := filepath.Join(dir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		switch field {
		case "worker_id":
			result.WorkerID = string(data)
		case "enrollment_credential":
			result.EnrollmentCredential = string(data)
		case "client_certificate":
			result.ClientCertificate = data
		case "client_key":
			result.ClientKey = data
		}
	}

	if result.WorkerID == "" || result.EnrollmentCredential == "" {
		return nil, fmt.Errorf("incomplete credentials")
	}

	return result, nil
}

// GetTLSConfig returns a TLS config using the enrolled client certificate.
func (c *EnrollClient) GetTLSConfig(caCertPEM []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(
		c.loadCert(),
		c.loadKey(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCertPEM)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (c *EnrollClient) loadCert() []byte {
	// Load from credentials directory
	path := filepath.Join(".worker-credentials", "client_certificate.pem")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

func (c *EnrollClient) loadKey() []byte {
	path := filepath.Join(".worker-credentials", "client_key.pem")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
