package workerclient

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

// Client implements the worker-facing protocol client over mTLS.
type Client struct {
	controlPlaneURL string
	workerID       string
	httpClient     *http.Client
}

// NewClient creates a new worker protocol client.
func NewClient(controlPlaneURL, workerID string, caCertPEM []byte, clientCert tls.Certificate) (*Client, error) {
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
	return &Client{
		controlPlaneURL: controlPlaneURL,
		workerID:       workerID,
		httpClient:     &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}, nil
}

// EstablishSession establishes a session using mTLS identity.
func (c *Client) EstablishSession() (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"worker_id": c.workerID,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/session",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("session request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("session failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		SessionToken string `json:"session_token"`
		ExpiresAt    int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode session response: %w", err)
	}

	return result.SessionToken, nil
}

// Heartbeat sends a heartbeat to the Control Plane.
func (c *Client) Heartbeat(status string) error {
	reqBody, _ := json.Marshal(map[string]string{
		"worker_id": c.workerID,
		"status":    status,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/heartbeat",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReportResources reports worker resource availability.
func (c *Client) ReportResources(cpuPercent float64, memoryMB, diskMB int64, activeJobs int32, gpuAvailable bool, gpuModel string) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"worker_id":     c.workerID,
		"cpu_percent":   cpuPercent,
		"memory_mb":     memoryMB,
		"disk_mb":       diskMB,
		"active_jobs":   activeJobs,
		"gpu_available": gpuAvailable,
		"gpu_model":     gpuModel,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/resources",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("resource report failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resource report failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SubmitResult submits a job result to the Control Plane.
func (c *Client) SubmitResult(jobID string, resultData []byte, exitCode int32, stdout, stderr string) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"job_id":      jobID,
		"worker_id":   c.workerID,
		"result_data": resultData,
		"exit_code":   exitCode,
		"stdout":      stdout,
		"stderr":      stderr,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/job/result",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("result submission failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("result submission failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateJobStatus updates the status of a job.
func (c *Client) UpdateJobStatus(jobID, status, message string) error {
	reqBody, _ := json.Marshal(map[string]string{
		"job_id":    jobID,
		"worker_id": c.workerID,
		"status":    status,
		"message":   message,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/job/status",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("status update failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status update failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// CancelJob requests job cancellation.
func (c *Client) CancelJob(jobID string) error {
	reqBody, _ := json.Marshal(map[string]string{
		"job_id":    jobID,
		"worker_id": c.workerID,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/job/cancel",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("cancel request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel request failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetEnvironment fetches environment content from the Control Plane.
func (c *Client) GetEnvironment(environmentID string) ([]byte, string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"environment_id": environmentID,
		"worker_id":      c.workerID,
	})

	resp, err := c.httpClient.Post(
		c.controlPlaneURL+"/v1/environment",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, "", fmt.Errorf("environment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("environment request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		EnvironmentID string `json:"environment_id"`
		Hash          string `json:"hash"`
		Content       []byte `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("failed to decode environment response: %w", err)
	}

	return result.Content, result.Hash, nil
}
