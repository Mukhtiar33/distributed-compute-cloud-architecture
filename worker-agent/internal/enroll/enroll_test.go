package enroll

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestEnrollResult_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	result := &EnrollResult{
		WorkerID:             "worker-test",
		EnrollmentCredential: "test-credential",
		ClientCertificate:    []byte("test-cert"),
		ClientKey:            []byte("test-key"),
	}

	client := &EnrollClient{}
	err := client.SaveCredentials(result, tmpDir)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Verify files exist
	files := []string{"worker_id", "enrollment_credential", "client_certificate.pem", "client_key.pem"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", path)
		}
	}

	// Load credentials
	loaded, err := client.LoadCredentials(tmpDir)
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if loaded.WorkerID != "worker-test" {
		t.Errorf("expected worker-test, got %s", loaded.WorkerID)
	}
	if loaded.EnrollmentCredential != "test-credential" {
		t.Errorf("expected test-credential, got %s", loaded.EnrollmentCredential)
	}
}

func TestEnrollResult_LoadIncomplete(t *testing.T) {
	tmpDir := t.TempDir()

	// Only write worker_id, not enrollment_credential
	os.WriteFile(filepath.Join(tmpDir, "worker_id"), []byte("worker-test"), 0600)

	client := &EnrollClient{}
	_, err := client.LoadCredentials(tmpDir)
	if err == nil {
		t.Error("expected error for incomplete credentials, got nil")
	}
}

func TestGenerateCSR(t *testing.T) {
	// Generate a keypair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	// Create CSR
	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "worker-test",
			Organization: []string{"Test"},
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privKey)
	if err != nil {
		t.Fatalf("failed to create CSR: %v", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	if len(csrPEM) == 0 {
		t.Error("CSR PEM should not be empty")
	}

	// Verify CSR can be parsed
	csrBlock, _ := pem.Decode(csrPEM)
	if csrBlock == nil {
		t.Fatal("failed to decode CSR PEM")
	}

	csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse CSR: %v", err)
	}

	if csr.Subject.CommonName != "worker-test" {
		t.Errorf("expected CN=worker-test, got %s", csr.Subject.CommonName)
	}
}
