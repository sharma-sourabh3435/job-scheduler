package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
	"github.com/sharma-sourabh3435/job-scheduler/internal/storage"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

// JobManager manages job assignment and queuing
type JobManager struct {
	storage storage.Storage
	logger  *utils.Logger
	queue   chan *models.JobRun
	mu      sync.RWMutex
}

// NewJobManager creates a new job manager instance
func NewJobManager(storage storage.Storage) *JobManager {
	return &JobManager{
		storage: storage,
		logger:  utils.NewLogger("job-manager", utils.INFO),
		queue:   make(chan *models.JobRun, 100),
	}
}

// Start starts the job manager
func (jm *JobManager) Start(ctx context.Context) error {
	jm.logger.Info("Starting job manager")

	// Load pending jobs from database on startup
	if err := jm.loadPendingJobs(ctx); err != nil {
		return fmt.Errorf("failed to load pending jobs: %w", err)
	}

	return nil
}

// loadPendingJobs loads pending jobs from the database into the queue
func (jm *JobManager) loadPendingJobs(ctx context.Context) error {
	pendingRuns, err := jm.storage.GetPendingJobRuns(ctx, 1000)
	if err != nil {
		return err
	}

	jm.logger.Info("Loading %d pending job runs", len(pendingRuns))

	for _, run := range pendingRuns {
		select {
		case jm.queue <- run:
			jm.logger.Debug("Queued job run %d (job %d)", run.RunID, run.JobID)
		case <-ctx.Done():
			return ctx.Err()
		default:
			jm.logger.Warn("Queue full, skipping job run %d", run.RunID)
		}
	}

	return nil
}

// EnqueueJobRun adds a job run to the queue
func (jm *JobManager) EnqueueJobRun(jobRun *models.JobRun) error {
	select {
	case jm.queue <- jobRun:
		jm.logger.Debug("Enqueued job run %d (job %d)", jobRun.RunID, jobRun.JobID)
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// GetNextJobRun retrieves the next job run from the queue (non-blocking)
func (jm *JobManager) GetNextJobRun() *models.JobRun {
	select {
	case jobRun := <-jm.queue:
		return jobRun
	default:
		return nil
	}
}

// GetNextJobRunBlocking retrieves the next job run from the queue (blocking with context)
func (jm *JobManager) GetNextJobRunBlocking(ctx context.Context) (*models.JobRun, error) {
	select {
	case jobRun := <-jm.queue:
		return jobRun, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetQueueSize returns the current queue size
func (jm *JobManager) GetQueueSize() int {
	return len(jm.queue)
}

// AssignJobToWorker assigns a job run to a worker
func (jm *JobManager) AssignJobToWorker(ctx context.Context, jobRun *models.JobRun, workerID string) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	// Update job run with worker assignment
	jobRun.WorkerID = workerID
	jobRun.Status = models.JobStatusRunning

	if err := jm.storage.UpdateJobRun(ctx, jobRun); err != nil {
		return fmt.Errorf("failed to assign job run %d to worker %s: %w", jobRun.RunID, workerID, err)
	}

	jm.logger.Info("Assigned job run %d (job %d) to worker %s", jobRun.RunID, jobRun.JobID, workerID)
	return nil
}

// RequeueJobRun requeues a failed or unassigned job run
func (jm *JobManager) RequeueJobRun(ctx context.Context, jobRun *models.JobRun) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	// Reset job run status
	jobRun.Status = models.JobStatusPending
	jobRun.WorkerID = ""

	if err := jm.storage.UpdateJobRun(ctx, jobRun); err != nil {
		return fmt.Errorf("failed to update job run status: %w", err)
	}

	// Add back to queue
	if err := jm.EnqueueJobRun(jobRun); err != nil {
		return fmt.Errorf("failed to requeue job run: %w", err)
	}

	jm.logger.Info("Requeued job run %d (job %d)", jobRun.RunID, jobRun.JobID)
	return nil
}

// GetJobForWorker retrieves a job for a specific worker (used by API)
func (jm *JobManager) GetJobForWorker(ctx context.Context, workerID string) (*models.Job, error) {
	// Get next job run from queue
	jobRun := jm.GetNextJobRun()
	if jobRun == nil {
		return nil, nil // No jobs available
	}

	// Assign to worker
	if err := jm.AssignJobToWorker(ctx, jobRun, workerID); err != nil {
		// If assignment fails, requeue the job
		jm.EnqueueJobRun(jobRun)
		return nil, err
	}

	// Get the full job details
	job, err := jm.storage.GetJob(ctx, jobRun.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}
