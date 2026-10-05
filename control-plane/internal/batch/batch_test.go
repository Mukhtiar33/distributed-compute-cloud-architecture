package batch

import (
	"testing"
	"time"
)

func TestManager_GroupInputs(t *testing.T) {
	m := NewManager()
	inputs := []string{"input1", "input2", "input3", "input4", "input5"}

	batches := m.GroupInputs("job-1", inputs, 2)

	if len(batches) != 2 {
		t.Errorf("expected 2 batches, got %d", len(batches))
	}

	// Check round-robin distribution
	if len(batches[0].Inputs) != 3 {
		t.Errorf("expected 3 inputs in batch 0, got %d", len(batches[0].Inputs))
	}
	if len(batches[1].Inputs) != 2 {
		t.Errorf("expected 2 inputs in batch 1, got %d", len(batches[1].Inputs))
	}
}

func TestManager_GroupInputsSingleWorker(t *testing.T) {
	m := NewManager()
	inputs := []string{"input1", "input2", "input3"}

	batches := m.GroupInputs("job-1", inputs, 1)

	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %d", len(batches))
	}
	if len(batches[0].Inputs) != 3 {
		t.Errorf("expected 3 inputs, got %d", len(batches[0].Inputs))
	}
}

func TestManager_UpdateStatus(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	if err := m.UpdateStatus(batchID, BatchStatusRunning); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	b, _ := m.GetBatch(batchID)
	if b.Status != BatchStatusRunning {
		t.Errorf("expected running, got %s", b.Status)
	}
	if b.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}
}

func TestManager_AssignWorker(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	if err := m.AssignWorker(batchID, "worker-1"); err != nil {
		t.Fatalf("AssignWorker failed: %v", err)
	}

	b, _ := m.GetBatch(batchID)
	if b.WorkerID != "worker-1" {
		t.Errorf("expected worker-1, got %s", b.WorkerID)
	}
	if b.Status != BatchStatusRunning {
		t.Errorf("expected running, got %s", b.Status)
	}
}

func TestManager_Complete(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	accepted, err := m.Complete(batchID, []byte("result"))
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if !accepted {
		t.Error("expected first completion to be accepted")
	}

	b, _ := m.GetBatch(batchID)
	if b.Status != BatchStatusSucceeded {
		t.Errorf("expected succeeded, got %s", b.Status)
	}
	if string(b.Result) != "result" {
		t.Errorf("expected result, got %s", string(b.Result))
	}
}

func TestManager_CompleteIdempotency(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	// First completion
	m.Complete(batchID, []byte("first result"))

	// Second completion (zombie/duplicate)
	accepted, err := m.Complete(batchID, []byte("second result"))
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if accepted {
		t.Error("expected duplicate completion to be rejected")
	}

	// Verify first result is preserved
	b, _ := m.GetBatch(batchID)
	if string(b.Result) != "first result" {
		t.Errorf("expected first result to be preserved, got %s", string(b.Result))
	}
}

func TestManager_Reassign(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	if err := m.Reassign(batchID, "worker-2"); err != nil {
		t.Fatalf("Reassign failed: %v", err)
	}

	b, _ := m.GetBatch(batchID)
	if b.Status != BatchStatusReassigned {
		t.Errorf("expected reassigned, got %s", b.Status)
	}
	if b.ReassignedTo != "worker-2" {
		t.Errorf("expected worker-2, got %s", b.ReassignedTo)
	}
	if b.ReassignCount != 1 {
		t.Errorf("expected reassign count 1, got %d", b.ReassignCount)
	}
}

func TestManager_IsJobComplete(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)

	// Not complete initially
	if m.IsJobComplete("job-1") {
		t.Error("expected job to not be complete initially")
	}

	// Complete all batches
	m.Complete("job-1-batch-0", []byte("result1"))
	m.Complete("job-1-batch-1", []byte("result2"))

	if !m.IsJobComplete("job-1") {
		t.Error("expected job to be complete after all batches succeed")
	}
}

func TestManager_IsJobCompletePartial(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)

	// Complete only one batch
	m.Complete("job-1-batch-0", []byte("result1"))

	if m.IsJobComplete("job-1") {
		t.Error("expected job to not be complete with only one batch done")
	}
}

func TestManager_GetBatchesByJob(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)
	m.GroupInputs("job-2", []string{"input3"}, 1)

	batches := m.GetBatchesByJob("job-1")
	if len(batches) != 2 {
		t.Errorf("expected 2 batches for job-1, got %d", len(batches))
	}
}

func TestManager_GetPendingBatches(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)

	pending := m.GetPendingBatches()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending batches, got %d", len(pending))
	}
}

func TestManager_GetRunningBatches(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)

	m.AssignWorker("job-1-batch-0", "worker-1")

	running := m.GetRunningBatches()
	if len(running) != 1 {
		t.Errorf("expected 1 running batch, got %d", len(running))
	}
}

func TestManager_GetFailedBatches(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)

	m.Fail("job-1-batch-0")

	failed := m.GetFailedBatches()
	if len(failed) != 1 {
		t.Errorf("expected 1 failed batch, got %d", len(failed))
	}
}

func TestManager_Stats(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2", "input3"}, 2)

	m.AssignWorker("job-1-batch-0", "worker-1")
	m.Complete("job-1-batch-1", []byte("result"))

	stats := m.Stats()
	if stats["total"] != 2 {
		t.Errorf("expected 2 total batches, got %d", stats["total"])
	}
	if stats["running"] != 1 {
		t.Errorf("expected 1 running batch, got %d", stats["running"])
	}
	if stats["succeeded"] != 1 {
		t.Errorf("expected 1 succeeded batch, got %d", stats["succeeded"])
	}
}

func TestManager_Fail(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	if err := m.Fail(batchID); err != nil {
		t.Fatalf("Fail failed: %v", err)
	}

	b, _ := m.GetBatch(batchID)
	if b.Status != BatchStatusFailed {
		t.Errorf("expected failed, got %s", b.Status)
	}
}

func TestManager_UpdateStatusNonexistent(t *testing.T) {
	m := NewManager()

	err := m.UpdateStatus("nonexistent", BatchStatusRunning)
	if err == nil {
		t.Error("expected error for nonexistent batch")
	}
}

func TestManager_CompleteNonexistent(t *testing.T) {
	m := NewManager()

	_, err := m.Complete("nonexistent", []byte("result"))
	if err == nil {
		t.Error("expected error for nonexistent batch")
	}
}

func TestManager_ReassignNonexistent(t *testing.T) {
	m := NewManager()

	err := m.Reassign("nonexistent", "worker-1")
	if err == nil {
		t.Error("expected error for nonexistent batch")
	}
}

func TestManager_ZombieResult(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	// Simulate: batch fails, gets reassigned, then original worker reports success
	m.AssignWorker(batchID, "worker-1")
	m.Fail(batchID)
	m.Reassign(batchID, "worker-2")

	// Original worker reports success (zombie)
	accepted, _ := m.Complete(batchID, []byte("zombie result"))

	// The batch was reassigned, so this completion should be accepted
	// (the new worker hasn't completed yet)
	if !accepted {
		t.Error("expected zombie result to be accepted (batch was reassigned)")
	}

	// Now the new worker also reports success (duplicate)
	accepted, _ = m.Complete(batchID, []byte("new worker result"))

	// This should be rejected (batch already succeeded)
	if accepted {
		t.Error("expected duplicate result to be rejected")
	}

	// Verify the first result is preserved
	b, _ := m.GetBatch(batchID)
	if string(b.Result) != "zombie result" {
		t.Errorf("expected zombie result to be preserved, got %s", string(b.Result))
	}
}

func TestManager_GroupInputsEmpty(t *testing.T) {
	m := NewManager()
	batches := m.GroupInputs("job-1", []string{}, 2)

	if len(batches) != 2 {
		t.Errorf("expected 2 batches even with no inputs, got %d", len(batches))
	}
}

func TestManager_GroupInputsMoreWorkersThanInputs(t *testing.T) {
	m := NewManager()
	inputs := []string{"input1"}

	batches := m.GroupInputs("job-1", inputs, 3)

	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d", len(batches))
	}

	// Only first batch should have inputs
	if len(batches[0].Inputs) != 1 {
		t.Errorf("expected 1 input in batch 0, got %d", len(batches[0].Inputs))
	}
	if len(batches[1].Inputs) != 0 {
		t.Errorf("expected 0 inputs in batch 1, got %d", len(batches[1].Inputs))
	}
	if len(batches[2].Inputs) != 0 {
		t.Errorf("expected 0 inputs in batch 2, got %d", len(batches[2].Inputs))
	}
}

func TestManager_Timestamps(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	b, _ := m.GetBatch(batchID)
	if b.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	m.AssignWorker(batchID, "worker-1")
	b, _ = m.GetBatch(batchID)
	if b.StartedAt == nil {
		t.Error("expected StartedAt to be set after assignment")
	}

	m.Complete(batchID, []byte("result"))
	b, _ = m.GetBatch(batchID)
	if b.CompletedAt == nil {
		t.Error("expected CompletedAt to be set after completion")
	}

	// Verify timestamp ordering
	if b.StartedAt.Before(b.CreatedAt) {
		t.Error("StartedAt should be after CreatedAt")
	}
	if b.CompletedAt.Before(*b.StartedAt) {
		t.Error("CompletedAt should be after StartedAt")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)

	// Simulate concurrent access
	done := make(chan bool, 2)

	go func() {
		m.AssignWorker("job-1-batch-0", "worker-1")
		done <- true
	}()

	go func() {
		m.AssignWorker("job-1-batch-1", "worker-2")
		done <- true
	}()

	<-done
	<-done

	// Both batches should be assigned
	b0, _ := m.GetBatch("job-1-batch-0")
	b1, _ := m.GetBatch("job-1-batch-1")

	if b0.WorkerID != "worker-1" {
		t.Errorf("expected worker-1, got %s", b0.WorkerID)
	}
	if b1.WorkerID != "worker-2" {
		t.Errorf("expected worker-2, got %s", b1.WorkerID)
	}
}

func TestManager_ReassignCount(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	// Reassign multiple times
	m.Reassign(batchID, "worker-2")
	m.Reassign(batchID, "worker-3")
	m.Reassign(batchID, "worker-4")

	b, _ := m.GetBatch(batchID)
	if b.ReassignCount != 3 {
		t.Errorf("expected reassign count 3, got %d", b.ReassignCount)
	}
	if b.ReassignedTo != "worker-4" {
		t.Errorf("expected worker-4, got %s", b.ReassignedTo)
	}
}

func TestManager_IsJobCompleteNoBatches(t *testing.T) {
	m := NewManager()

	if m.IsJobComplete("nonexistent-job") {
		t.Error("expected job with no batches to not be complete")
	}
}

func TestManager_GetBatchesByJobNonexistent(t *testing.T) {
	m := NewManager()

	batches := m.GetBatchesByJob("nonexistent")
	if len(batches) != 0 {
		t.Errorf("expected 0 batches for nonexistent job, got %d", len(batches))
	}
}

func TestManager_StatusTransitions(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	// pending -> running
	m.AssignWorker(batchID, "worker-1")
	b, _ := m.GetBatch(batchID)
	if b.Status != BatchStatusRunning {
		t.Errorf("expected running, got %s", b.Status)
	}

	// running -> failed
	m.Fail(batchID)
	b, _ = m.GetBatch(batchID)
	if b.Status != BatchStatusFailed {
		t.Errorf("expected failed, got %s", b.Status)
	}

	// failed -> reassigned
	m.Reassign(batchID, "worker-2")
	b, _ = m.GetBatch(batchID)
	if b.Status != BatchStatusReassigned {
		t.Errorf("expected reassigned, got %s", b.Status)
	}

	// reassigned -> running
	m.AssignWorker(batchID, "worker-2")
	b, _ = m.GetBatch(batchID)
	if b.Status != BatchStatusRunning {
		t.Errorf("expected running, got %s", b.Status)
	}

	// running -> succeeded
	m.Complete(batchID, []byte("result"))
	b, _ = m.GetBatch(batchID)
	if b.Status != BatchStatusSucceeded {
		t.Errorf("expected succeeded, got %s", b.Status)
	}
}

func TestManager_ZombieAfterSuccess(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	// Complete the batch
	m.Complete(batchID, []byte("first result"))

	// Zombie worker reports success after batch already succeeded
	accepted, _ := m.Complete(batchID, []byte("zombie result"))

	if accepted {
		t.Error("expected zombie result to be rejected after batch succeeded")
	}

	// Verify original result is preserved
	b, _ := m.GetBatch(batchID)
	if string(b.Result) != "first result" {
		t.Errorf("expected first result to be preserved, got %s", string(b.Result))
	}
}

func TestManager_MultipleJobs(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1", "input2"}, 2)
	m.GroupInputs("job-2", []string{"input3", "input4"}, 2)

	// Complete job-1
	m.Complete("job-1-batch-0", []byte("r1"))
	m.Complete("job-1-batch-1", []byte("r2"))

	// job-1 should be complete, job-2 should not
	if !m.IsJobComplete("job-1") {
		t.Error("expected job-1 to be complete")
	}
	if m.IsJobComplete("job-2") {
		t.Error("expected job-2 to not be complete")
	}
}

func TestManager_StatsEmpty(t *testing.T) {
	m := NewManager()

	stats := m.Stats()
	if stats["total"] != 0 {
		t.Errorf("expected 0 total batches, got %d", stats["total"])
	}
}

func TestManager_PendingAfterReassign(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	// Reassign the batch
	m.Reassign(batchID, "worker-2")

	// Reassigned batches should appear in pending
	pending := m.GetPendingBatches()
	if len(pending) != 1 {
		t.Errorf("expected 1 pending batch after reassignment, got %d", len(pending))
	}
}

func TestManager_TimeTracking(t *testing.T) {
	m := NewManager()
	m.GroupInputs("job-1", []string{"input1"}, 1)

	batchID := "job-1-batch-0"

	b, _ := m.GetBatch(batchID)
	createdAt := b.CreatedAt

	// Ensure CreatedAt is recent
	if time.Since(createdAt) > time.Minute {
		t.Error("CreatedAt should be recent")
	}

	m.AssignWorker(batchID, "worker-1")
	b, _ = m.GetBatch(batchID)

	if b.StartedAt == nil {
		t.Fatal("StartedAt should be set")
	}

	// StartedAt should be >= CreatedAt
	if b.StartedAt.Before(createdAt) {
		t.Error("StartedAt should be >= CreatedAt")
	}
}
