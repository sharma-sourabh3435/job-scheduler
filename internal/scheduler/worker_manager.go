package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
	"github.com/sharma-sourabh3435/job-scheduler/internal/storage"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

// WorkerManager manages worker nodes and their health
type WorkerManager struct {
	storage         storage.Storage
	logger          *utils.Logger
	workers         map[string]*models.Worker
	mu              sync.RWMutex
	timeoutDuration time.Duration
}

// NewWorkerManager creates a new worker manager instance
func NewWorkerManager(storage storage.Storage, timeoutDuration time.Duration) *WorkerManager {
	return &WorkerManager{
		storage:         storage,
		logger:          utils.NewLogger("worker-manager", utils.INFO),
		workers:         make(map[string]*models.Worker),
		timeoutDuration: timeoutDuration,
	}
}

// Start starts the worker manager
func (wm *WorkerManager) Start(ctx context.Context) error {
	wm.logger.Info("Starting worker manager")

	// Load workers from database
	if err := wm.loadWorkers(ctx); err != nil {
		return err
	}

	// Start heartbeat monitor
	go wm.monitorHeartbeats(ctx)

	return nil
}

// loadWorkers loads workers from the database
func (wm *WorkerManager) loadWorkers(ctx context.Context) error {
	workers, err := wm.storage.ListWorkers(ctx)
	if err != nil {
		return err
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, worker := range workers {
		wm.workers[worker.WorkerID] = worker
	}

	wm.logger.Info("Loaded %d workers from database", len(workers))
	return nil
}

// monitorHeartbeats continuously checks worker heartbeats
func (wm *WorkerManager) monitorHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wm.logger.Debug("Heartbeat monitor stopped")
			return
		case <-ticker.C:
			wm.checkWorkerHealth(ctx)
		}
	}
}

// checkWorkerHealth checks all workers for timeout
func (wm *WorkerManager) checkWorkerHealth(ctx context.Context) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()

	for workerID, worker := range wm.workers {
		if worker.Status == models.WorkerStatusActive {
			timeSinceLastSeen := now.Sub(worker.LastSeen)

			if timeSinceLastSeen > wm.timeoutDuration {
				wm.logger.Warn("Worker %s timed out (last seen %v ago)", workerID, timeSinceLastSeen)

				// Mark worker as inactive
				worker.Status = models.WorkerStatusInactive
				if err := wm.storage.UpdateWorkerStatus(ctx, workerID, models.WorkerStatusInactive); err != nil {
					wm.logger.Error("Failed to update worker %s status: %v", workerID, err)
				}
			}
		}
	}
}

// RegisterWorker registers or updates a worker
func (wm *WorkerManager) RegisterWorker(ctx context.Context, worker *models.Worker) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.workers[worker.WorkerID] = worker
	wm.logger.Info("Registered worker: %s", worker.WorkerID)

	return nil
}

// UpdateWorkerHeartbeat updates a worker's last seen time
func (wm *WorkerManager) UpdateWorkerHeartbeat(workerID string, lastSeen time.Time) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if worker, exists := wm.workers[workerID]; exists {
		worker.LastSeen = lastSeen
		worker.Status = models.WorkerStatusActive
	}
}

// GetActiveWorkers returns all active workers
func (wm *WorkerManager) GetActiveWorkers() []*models.Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var activeWorkers []*models.Worker
	for _, worker := range wm.workers {
		if worker.Status == models.WorkerStatusActive {
			activeWorkers = append(activeWorkers, worker)
		}
	}

	return activeWorkers
}

// GetAvailableWorker returns an available worker (simple round-robin for MVP)
func (wm *WorkerManager) GetAvailableWorker() *models.Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	for _, worker := range wm.workers {
		if worker.Status == models.WorkerStatusActive {
			return worker
		}
	}

	return nil
}

// GetWorker returns a specific worker
func (wm *WorkerManager) GetWorker(workerID string) *models.Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.workers[workerID]
}

// GetWorkerCount returns the total number of workers
func (wm *WorkerManager) GetWorkerCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return len(wm.workers)
}

// GetActiveWorkerCount returns the number of active workers
func (wm *WorkerManager) GetActiveWorkerCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	count := 0
	for _, worker := range wm.workers {
		if worker.Status == models.WorkerStatusActive {
			count++
		}
	}

	return count
}

// HandleWorkerFailure handles a failed worker by requeueing its jobs
func (wm *WorkerManager) HandleWorkerFailure(ctx context.Context, workerID string, jobManager *JobManager) error {
	wm.logger.Warn("Handling failure for worker %s", workerID)

	// Get running jobs for this worker
	runningJobs, err := wm.storage.GetRunningJobRunsByWorker(ctx, workerID)
	if err != nil {
		return err
	}

	wm.logger.Info("Found %d running jobs for failed worker %s", len(runningJobs), workerID)

	// Requeue all running jobs
	for _, jobRun := range runningJobs {
		if err := jobManager.RequeueJobRun(ctx, jobRun); err != nil {
			wm.logger.Error("Failed to requeue job run %d: %v", jobRun.RunID, err)
		}
	}

	return nil
}
