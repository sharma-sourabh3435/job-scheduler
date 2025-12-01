package storage

import (
	"context"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
)

// Storage defines the interface for database operations
type Storage interface {
	// Job operations
	CreateJob(ctx context.Context, job *models.Job) error
	GetJob(ctx context.Context, id int) (*models.Job, error)
	ListJobs(ctx context.Context, limit, offset int) ([]*models.Job, error)
	UpdateJob(ctx context.Context, job *models.Job) error
	DeleteJob(ctx context.Context, id int) error

	// JobRun operations
	CreateJobRun(ctx context.Context, jobRun *models.JobRun) error
	GetJobRun(ctx context.Context, runID int) (*models.JobRun, error)
	UpdateJobRun(ctx context.Context, jobRun *models.JobRun) error
	GetJobRunsByJobID(ctx context.Context, jobID int, limit, offset int) ([]*models.JobRun, error)
	GetLastJobRun(ctx context.Context, jobID int) (*models.JobRun, error)
	GetPendingJobRuns(ctx context.Context, limit int) ([]*models.JobRun, error)
	GetRunningJobRunsByWorker(ctx context.Context, workerID string) ([]*models.JobRun, error)

	// Worker operations
	RegisterWorker(ctx context.Context, worker *models.Worker) error
	GetWorker(ctx context.Context, workerID string) (*models.Worker, error)
	UpdateWorkerHeartbeat(ctx context.Context, workerID string, lastSeen time.Time) error
	UpdateWorkerStatus(ctx context.Context, workerID string, status string) error
	GetActiveWorkers(ctx context.Context) ([]*models.Worker, error)
	ListWorkers(ctx context.Context) ([]*models.Worker, error)

	// Database management
	Close() error
	Ping(ctx context.Context) error
}
