package models

import "time"

// JobRun represents an execution instance of a job
type JobRun struct {
	RunID     int        `json:"run_id" db:"run_id"`
	JobID     int        `json:"job_id" db:"job_id"`
	WorkerID  string     `json:"worker_id,omitempty" db:"worker_id"`
	Status    string     `json:"status" db:"status"`
	StartTime *time.Time `json:"start_time,omitempty" db:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"`
	Logs      string     `json:"logs,omitempty" db:"logs"`
	ExitCode  *int       `json:"exit_code,omitempty" db:"exit_code"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// UpdateJobRunRequest represents the request to update a job run with logs and status
type UpdateJobRunRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	Status   string `json:"status" binding:"required"`
	Logs     string `json:"logs,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}
