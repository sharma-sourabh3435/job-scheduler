package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/storage"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

// Scheduler is the main scheduler that coordinates job assignment
type Scheduler struct {
	storage            storage.Storage
	jobManager         *JobManager
	workerManager      *WorkerManager
	logger             *utils.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	assignmentInterval time.Duration
}

// Config holds scheduler configuration
type Config struct {
	Storage            storage.Storage
	AssignmentInterval time.Duration
	WorkerTimeout      time.Duration
}

// NewScheduler creates a new scheduler instance
func NewScheduler(config Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		storage:            config.Storage,
		jobManager:         NewJobManager(config.Storage),
		workerManager:      NewWorkerManager(config.Storage, config.WorkerTimeout),
		logger:             utils.NewLogger("scheduler", utils.INFO),
		ctx:                ctx,
		cancel:             cancel,
		assignmentInterval: config.AssignmentInterval,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.logger.Info("Starting scheduler")

	// Start job manager
	if err := s.jobManager.Start(s.ctx); err != nil {
		return err
	}

	// Start worker manager
	if err := s.workerManager.Start(s.ctx); err != nil {
		return err
	}

	// Start job assignment loop
	s.wg.Add(1)
	go s.assignmentLoop()

	s.logger.Info("Scheduler started successfully")
	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping scheduler")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("Scheduler stopped")
}

// assignmentLoop continuously assigns jobs to available workers
func (s *Scheduler) assignmentLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.assignmentInterval)
	defer ticker.Stop()

	s.logger.Info("Job assignment loop started (interval: %v)", s.assignmentInterval)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Assignment loop stopped")
			return
		case <-ticker.C:
			s.tryAssignJobs()
		}
	}
}

// tryAssignJobs attempts to assign pending jobs to available workers
func (s *Scheduler) tryAssignJobs() {
	// Check if there are jobs in the queue
	queueSize := s.jobManager.GetQueueSize()
	if queueSize == 0 {
		return
	}

	// Check if there are active workers
	activeWorkerCount := s.workerManager.GetActiveWorkerCount()
	if activeWorkerCount == 0 {
		s.logger.Debug("No active workers available for job assignment")
		return
	}

	s.logger.Debug("Attempting job assignment (%d jobs queued, %d active workers)", queueSize, activeWorkerCount)

	// Try to assign jobs
	assigned := 0
	maxAssignments := min(queueSize, activeWorkerCount*2) // Limit assignments per cycle

	for i := 0; i < maxAssignments; i++ {
		// Get next job
		jobRun := s.jobManager.GetNextJobRun()
		if jobRun == nil {
			break
		}

		// Get available worker
		worker := s.workerManager.GetAvailableWorker()
		if worker == nil {
			// No workers available, put job back in queue
			s.jobManager.EnqueueJobRun(jobRun)
			break
		}

		// Assign job to worker
		if err := s.jobManager.AssignJobToWorker(s.ctx, jobRun, worker.WorkerID); err != nil {
			s.logger.Error("Failed to assign job run %d to worker %s: %v", jobRun.RunID, worker.WorkerID, err)
			// Put job back in queue
			s.jobManager.EnqueueJobRun(jobRun)
			continue
		}

		assigned++
		s.logger.Info("Assigned job run %d (job %d) to worker %s", jobRun.RunID, jobRun.JobID, worker.WorkerID)
	}

	if assigned > 0 {
		s.logger.Info("Assigned %d jobs in this cycle", assigned)
	}
}

// GetJobManager returns the job manager
func (s *Scheduler) GetJobManager() *JobManager {
	return s.jobManager
}

// GetWorkerManager returns the worker manager
func (s *Scheduler) GetWorkerManager() *WorkerManager {
	return s.workerManager
}

// GetStats returns current scheduler statistics
func (s *Scheduler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"queue_size":          s.jobManager.GetQueueSize(),
		"total_workers":       s.workerManager.GetWorkerCount(),
		"active_workers":      s.workerManager.GetActiveWorkerCount(),
		"assignment_interval": s.assignmentInterval.String(),
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
