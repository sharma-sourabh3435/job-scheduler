package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
)

func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	// Create temporary database file
	tmpFile := "test_scheduler.db"

	storage, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		storage.Close()
		os.Remove(tmpFile)
	}

	return storage, cleanup
}

func TestCreateAndGetJob(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job
	job := &models.Job{
		Name:     "test-job",
		Command:  "echo hello",
		Schedule: "*/5 * * * *",
	}

	err := storage.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	if job.ID == 0 {
		t.Error("Expected job ID to be set")
	}

	// Get the job
	retrieved, err := storage.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.Name != job.Name || retrieved.Command != job.Command {
		t.Errorf("Retrieved job doesn't match. Got %+v, want %+v", retrieved, job)
	}
}

func TestListJobs(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		job := &models.Job{
			Name:    "test-job",
			Command: "echo hello",
		}
		if err := storage.CreateJob(ctx, job); err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}
	}

	// List jobs
	jobs, err := storage.ListJobs(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}
}

func TestCreateAndUpdateJobRun(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job first
	job := &models.Job{
		Name:    "test-job",
		Command: "echo hello",
	}
	if err := storage.CreateJob(ctx, job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create a job run
	now := time.Now()
	jobRun := &models.JobRun{
		JobID:     job.ID,
		WorkerID:  "worker-1",
		Status:    models.JobStatusPending,
		StartTime: &now,
	}

	err := storage.CreateJobRun(ctx, jobRun)
	if err != nil {
		t.Fatalf("Failed to create job run: %v", err)
	}

	if jobRun.RunID == 0 {
		t.Error("Expected run ID to be set")
	}

	// Update job run
	exitCode := 0
	endTime := time.Now()
	jobRun.Status = models.JobStatusSuccess
	jobRun.EndTime = &endTime
	jobRun.Logs = "Job completed successfully"
	jobRun.ExitCode = &exitCode

	err = storage.UpdateJobRun(ctx, jobRun)
	if err != nil {
		t.Fatalf("Failed to update job run: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetJobRun(ctx, jobRun.RunID)
	if err != nil {
		t.Fatalf("Failed to get job run: %v", err)
	}

	if retrieved.Status != models.JobStatusSuccess {
		t.Errorf("Expected status %s, got %s", models.JobStatusSuccess, retrieved.Status)
	}
}

func TestRegisterAndGetWorker(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Register worker
	worker := &models.Worker{
		WorkerID: "worker-1",
	}

	err := storage.RegisterWorker(ctx, worker)
	if err != nil {
		t.Fatalf("Failed to register worker: %v", err)
	}

	if worker.Status != models.WorkerStatusActive {
		t.Errorf("Expected worker status to be active, got %s", worker.Status)
	}

	// Get worker
	retrieved, err := storage.GetWorker(ctx, worker.WorkerID)
	if err != nil {
		t.Fatalf("Failed to get worker: %v", err)
	}

	if retrieved.WorkerID != worker.WorkerID {
		t.Errorf("Expected worker ID %s, got %s", worker.WorkerID, retrieved.WorkerID)
	}
}

func TestUpdateWorkerHeartbeat(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Register worker
	worker := &models.Worker{
		WorkerID: "worker-1",
	}
	if err := storage.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("Failed to register worker: %v", err)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Update heartbeat
	newTime := time.Now()
	err := storage.UpdateWorkerHeartbeat(ctx, worker.WorkerID, newTime)
	if err != nil {
		t.Fatalf("Failed to update heartbeat: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetWorker(ctx, worker.WorkerID)
	if err != nil {
		t.Fatalf("Failed to get worker: %v", err)
	}

	if retrieved.LastSeen.Before(worker.LastSeen) {
		t.Error("Expected last seen to be updated")
	}
}

func TestGetActiveWorkers(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Register multiple workers
	for i := 1; i <= 3; i++ {
		worker := &models.Worker{
			WorkerID: "worker-" + string(rune('0'+i)),
		}
		if err := storage.RegisterWorker(ctx, worker); err != nil {
			t.Fatalf("Failed to register worker: %v", err)
		}
	}

	// Get active workers
	workers, err := storage.GetActiveWorkers(ctx)
	if err != nil {
		t.Fatalf("Failed to get active workers: %v", err)
	}

	if len(workers) != 3 {
		t.Errorf("Expected 3 active workers, got %d", len(workers))
	}
}

func TestGetPendingJobRuns(t *testing.T) {
	storage, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job
	job := &models.Job{
		Name:    "test-job",
		Command: "echo hello",
	}
	if err := storage.CreateJob(ctx, job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create pending job runs
	for i := 0; i < 3; i++ {
		jobRun := &models.JobRun{
			JobID:  job.ID,
			Status: models.JobStatusPending,
		}
		if err := storage.CreateJobRun(ctx, jobRun); err != nil {
			t.Fatalf("Failed to create job run: %v", err)
		}
	}

	// Get pending runs
	runs, err := storage.GetPendingJobRuns(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending job runs: %v", err)
	}

	if len(runs) != 3 {
		t.Errorf("Expected 3 pending runs, got %d", len(runs))
	}
}
