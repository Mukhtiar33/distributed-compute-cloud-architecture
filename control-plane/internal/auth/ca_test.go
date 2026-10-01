package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCA_SignCSR(t *testing.T) {
	// Create temp dir for test certs
	tmpDir := t.TempDir()

	// Generate CA key and cert
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	// Write CA cert and key to temp dir
	caCertPath := filepath.Join(tmpDir, "ca.crt")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	os.WriteFile(caCertPath, caCertPEM, 0644)
	os.WriteFile(caKeyPath, caKeyPEM, 0600)

	// Load CA
	ca, err := NewCA(caCertPath, caKeyPath)
	if err != nil {
		t.Fatalf("NewCA failed: %v", err)
	}

	// Generate worker key and CSR
	workerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate worker key: %v", err)
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "worker-1",
			Organization: []string{"Test"},
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, workerKey)
	if err != nil {
		t.Fatalf("failed to create CSR: %v", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	// Sign CSR
	clientCertPEM, err := ca.SignCSR(csrPEM, "worker-1")
	if err != nil {
		t.Fatalf("SignCSR failed: %v", err)
	}

	// Verify the signed cert
	certBlock, _ := pem.Decode(clientCertPEM)
	if certBlock == nil {
		t.Fatal("failed to decode client cert PEM")
	}

	clientCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse client cert: %v", err)
	}

	if clientCert.Subject.CommonName != "worker-1" {
		t.Errorf("expected CN=worker-1, got %s", clientCert.Subject.CommonName)
	}

	// Verify cert is signed by CA
	caCertBlock, _ := pem.Decode(caCertPEM)
	caCert, _ := x509.ParseCertificate(caCertBlock.Bytes)

	if err := clientCert.CheckSignatureFrom(caCert); err != nil {
		t.Errorf("cert signature verification failed: %v", err)
	}
}
