package models

import "time"

// Job represents a job definition in the scheduler
type Job struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Command   string    `json:"command" db:"command"`
	Schedule  string    `json:"schedule,omitempty" db:"schedule"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateJobRequest represents the request payload for creating a new job
type CreateJobRequest struct {
	Name     string `json:"name" binding:"required"`
	Command  string `json:"command" binding:"required"`
	Schedule string `json:"schedule,omitempty"`
}

// JobWithLastRun represents a job with its most recent run information
type JobWithLastRun struct {
	Job
	LastRun *JobRun `json:"last_run,omitempty"`
}
