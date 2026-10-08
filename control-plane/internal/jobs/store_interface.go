package jobs

// JobStoreInterface defines the job storage contract.
type JobStoreInterface interface {
	AddJob(job *Job)
	GetJob(id string) (*Job, bool)
	UpdateStatus(id string, status JobStatus) error
	ListJobs() []*Job
	GetJobsByStatus(status JobStatus) []*Job
}

// Ensure both implementations satisfy the interface
var _ JobStoreInterface = (*Store)(nil)
var _ JobStoreInterface = (*StoreDB)(nil)
