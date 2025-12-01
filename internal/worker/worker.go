package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

// Worker represents a job worker
type Worker struct {
	id                string
	schedulerURL      string
	pollInterval      time.Duration
	heartbeatInterval time.Duration
	logger            *utils.Logger
	executor          *Executor
	httpClient        *http.Client
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	currentJob        *models.Job
	currentJobMutex   sync.RWMutex
}

// Config holds worker configuration
type Config struct {
	WorkerID          string
	SchedulerURL      string
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
}

// NewWorker creates a new worker instance
func NewWorker(config Config) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	logger := utils.NewLogger(fmt.Sprintf("worker-%s", config.WorkerID), utils.INFO)

	return &Worker{
		id:                config.WorkerID,
		schedulerURL:      config.SchedulerURL,
		pollInterval:      config.PollInterval,
		heartbeatInterval: config.HeartbeatInterval,
		logger:            logger,
		executor:          NewExecutor(logger),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the worker
func (w *Worker) Start() error {
	w.logger.Info("Starting worker %s", w.id)

	// Register with scheduler
	if err := w.register(); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// Start heartbeat goroutine
	w.wg.Add(1)
	go w.heartbeatLoop()

	// Start polling goroutine
	w.wg.Add(1)
	go w.pollingLoop()

	// Wait for shutdown signal
	w.waitForShutdown()

	return nil
}

// Stop stops the worker gracefully
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker %s", w.id)
	w.cancel()
	w.wg.Wait()
	w.logger.Info("Worker %s stopped", w.id)
}

// register registers the worker with the scheduler
func (w *Worker) register() error {
	w.logger.Info("Registering with scheduler at %s", w.schedulerURL)

	reqBody := models.RegisterWorkerRequest{
		WorkerID: w.id,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/workers/register", w.schedulerURL)
	resp, err := w.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	w.logger.Info("Successfully registered with scheduler")
	return nil
}

// heartbeatLoop sends periodic heartbeats to the scheduler
func (w *Worker) heartbeatLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug("Heartbeat loop stopped")
			return
		case <-ticker.C:
			if err := w.sendHeartbeat(); err != nil {
				w.logger.Error("Failed to send heartbeat: %v", err)
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the scheduler
func (w *Worker) sendHeartbeat() error {
	reqBody := models.HeartbeatRequest{
		WorkerID: w.id,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/workers/heartbeat", w.schedulerURL)
	resp, err := w.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	w.logger.Debug("Heartbeat sent successfully")
	return nil
}

// pollingLoop continuously polls for new jobs
func (w *Worker) pollingLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug("Polling loop stopped")
			return
		case <-ticker.C:
			w.pollForJob()
		}
	}
}

// pollForJob polls the scheduler for the next job
func (w *Worker) pollForJob() {
	url := fmt.Sprintf("%s/workers/%s/next", w.schedulerURL, w.id)

	resp, err := w.httpClient.Get(url)
	if err != nil {
		w.logger.Error("Failed to poll for job: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logger.Warn("Poll returned status %d", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		w.logger.Error("Failed to decode response: %v", err)
		return
	}

	// Check if there's a job
	jobData, ok := result["job"]
	if !ok || jobData == nil {
		w.logger.Debug("No job available")
		return
	}

	// Parse job
	jobBytes, _ := json.Marshal(jobData)
	var job models.Job
	if err := json.Unmarshal(jobBytes, &job); err != nil {
		w.logger.Error("Failed to parse job: %v", err)
		return
	}

	w.logger.Info("Received job %d: %s", job.ID, job.Name)
	w.executeJob(&job)
}

// executeJob executes a job
func (w *Worker) executeJob(job *models.Job) {
	w.currentJobMutex.Lock()
	w.currentJob = job
	w.currentJobMutex.Unlock()

	defer func() {
		w.currentJobMutex.Lock()
		w.currentJob = nil
		w.currentJobMutex.Unlock()
	}()

	// Update status to running
	w.updateJobStatus(job.ID, models.JobStatusRunning, "", nil)

	// Execute the job
	w.logger.Info("Executing job %d: %s", job.ID, job.Command)
	result := w.executor.Execute(w.ctx, job.Command)

	// Update final status
	status := models.JobStatusSuccess
	if result.ExitCode != 0 {
		status = models.JobStatusFailed
	}

	logs := fmt.Sprintf("STDOUT:\n%s\n\nSTDERR:\n%s", result.Stdout, result.Stderr)
	w.updateJobStatus(job.ID, status, logs, &result.ExitCode)

	w.logger.Info("Job %d completed with status %s (exit code: %d)", job.ID, status, result.ExitCode)
}

// updateJobStatus updates the job status on the scheduler
func (w *Worker) updateJobStatus(jobID int, status string, logs string, exitCode *int) {
	reqBody := models.UpdateJobRunRequest{
		WorkerID: w.id,
		Status:   status,
		Logs:     logs,
		ExitCode: exitCode,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		w.logger.Error("Failed to marshal update request: %v", err)
		return
	}

	url := fmt.Sprintf("%s/jobs/%d/log", w.schedulerURL, jobID)
	resp, err := w.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		w.logger.Error("Failed to update job status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		w.logger.Error("Failed to update job status: %d - %s", resp.StatusCode, string(bodyBytes))
		return
	}

	w.logger.Debug("Updated job %d status to %s", jobID, status)
}

// waitForShutdown waits for shutdown signal
func (w *Worker) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	w.logger.Info("Received shutdown signal")

	// Wait for current job to complete (with timeout)
	w.currentJobMutex.RLock()
	hasCurrentJob := w.currentJob != nil
	w.currentJobMutex.RUnlock()

	if hasCurrentJob {
		w.logger.Info("Waiting for current job to complete...")
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				w.logger.Warn("Timeout waiting for job completion, forcing shutdown")
				w.Stop()
				return
			case <-ticker.C:
				w.currentJobMutex.RLock()
				stillRunning := w.currentJob != nil
				w.currentJobMutex.RUnlock()

				if !stillRunning {
					w.Stop()
					return
				}
			}
		}
	}

	w.Stop()
}
