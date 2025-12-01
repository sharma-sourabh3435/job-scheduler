package models

// Job status constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusSuccess   = "success"
	JobStatusFailed    = "failed"
	JobStatusCancelled = "cancelled"
)

// Worker status constants
const (
	WorkerStatusActive   = "active"
	WorkerStatusInactive = "inactive"
)

// Default configuration values
const (
	DefaultHeartbeatInterval = 10 // seconds
	DefaultPollInterval      = 5  // seconds
	DefaultWorkerTimeout     = 30 // seconds - mark worker inactive after this
)
