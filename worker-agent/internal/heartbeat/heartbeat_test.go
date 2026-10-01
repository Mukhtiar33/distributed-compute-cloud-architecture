package heartbeat

import (
	"testing"
	"time"
)

func TestHeartbeatClient_StartStop(t *testing.T) {
	client := &HeartbeatClient{
		controlPlaneURL: "https://localhost:8443",
		workerID:       "worker-test",
		sessionToken:   "test-token",
		interval:       1 * time.Second,
		stopCh:         make(chan struct{}),
	}

	client.Start()
	if !client.running {
		t.Error("client should be running after Start")
	}

	client.Stop()
	if client.running {
		t.Error("client should not be running after Stop")
	}
}

func TestHeartbeatClient_UpdateSessionToken(t *testing.T) {
	client := &HeartbeatClient{
		workerID: "worker-test",
	}

	client.UpdateSessionToken("new-token")

	client.mu.RLock()
	token := client.sessionToken
	client.mu.RUnlock()

	if token != "new-token" {
		t.Errorf("expected new-token, got %s", token)
	}
}
