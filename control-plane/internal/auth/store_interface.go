package auth

// WorkerStoreInterface defines the worker storage contract.
type WorkerStoreInterface interface {
	AddWorker(w *Worker)
	GetWorker(id string) (*Worker, bool)
	UpdateWorkerStatus(id, status string) bool
	RevokeWorker(id string) bool
	ListWorkers() []*Worker
	IsWorkerRevoked(id string) bool
}

// Ensure both implementations satisfy the interface
var _ WorkerStoreInterface = (*WorkerStore)(nil)
var _ WorkerStoreInterface = (*WorkerStoreDB)(nil)
