package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
	"github.com/sharma-sourabh3435/job-scheduler/internal/scheduler"
	"github.com/sharma-sourabh3435/job-scheduler/internal/storage"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

// Server represents the API server
type Server struct {
	storage    storage.Storage
	jobManager *scheduler.JobManager
	logger     *utils.Logger
	server     *http.Server
}

// NewServer creates a new API server instance
func NewServer(storage storage.Storage, jobManager *scheduler.JobManager, addr string) *Server {
	logger := utils.NewLogger("api", utils.INFO)

	s := &Server{
		storage:    storage,
		jobManager: jobManager,
		logger:     logger,
	}

	mux := http.NewServeMux()

	// Job endpoints
	mux.HandleFunc("/jobs", s.loggingMiddleware(s.handleJobs))
	mux.HandleFunc("/jobs/", s.loggingMiddleware(s.handleJobByID))

	// Worker endpoints
	mux.HandleFunc("/workers/register", s.loggingMiddleware(s.handleWorkerRegister))
	mux.HandleFunc("/workers/heartbeat", s.loggingMiddleware(s.handleWorkerHeartbeat))
	mux.HandleFunc("/workers/", s.loggingMiddleware(s.handleWorkerNext))

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the API server
func (s *Server) Start() error {
	s.logger.Info("Starting API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

// Middleware: CORS
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware: Logging
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Debug("%s %s", r.Method, r.URL.Path)

		next(w, r)

		s.logger.Debug("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	}
}

// Helper: JSON response
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response: %v", err)
	}
}

// Helper: Error response
func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// Handler: Jobs (POST /jobs, GET /jobs)
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createJob(w, r)
	case http.MethodGet:
		s.listJobs(w, r)
	default:
		s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// Handler: Create job
func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req models.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Command == "" {
		s.errorResponse(w, http.StatusBadRequest, "Name and command are required")
		return
	}

	job := &models.Job{
		Name:     req.Name,
		Command:  req.Command,
		Schedule: req.Schedule,
	}

	if err := s.storage.CreateJob(r.Context(), job); err != nil {
		s.logger.Error("Failed to create job: %v", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to create job")
		return
	}

	// Create initial job run in pending state
	jobRun := &models.JobRun{
		JobID:  job.ID,
		Status: models.JobStatusPending,
	}
	if err := s.storage.CreateJobRun(r.Context(), jobRun); err != nil {
		s.logger.Error("Failed to create job run: %v", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to create job run")
		return
	}

	// Enqueue the job run
	if s.jobManager != nil {
		if err := s.jobManager.EnqueueJobRun(jobRun); err != nil {
			s.logger.Error("Failed to enqueue job run: %v", err)
		}
	}

	s.logger.Info("Created job: %d - %s", job.ID, job.Name)
	s.jsonResponse(w, http.StatusCreated, job)
}

// Handler: List jobs
func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	jobs, err := s.storage.ListJobs(r.Context(), limit, offset)
	if err != nil {
		s.logger.Error("Failed to list jobs: %v", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to list jobs")
		return
	}

	s.jsonResponse(w, http.StatusOK, jobs)
}

// Handler: Job by ID (GET /jobs/{id}, POST /jobs/{id}/log)
func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/jobs/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		s.errorResponse(w, http.StatusBadRequest, "Job ID required")
		return
	}

	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Handle /jobs/{id}/log
	if len(parts) == 2 && parts[1] == "log" {
		if r.Method == http.MethodPost {
			s.updateJobLog(w, r, jobID)
		} else {
			s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	// Handle /jobs/{id}
	if r.Method == http.MethodGet {
		s.getJob(w, r, jobID)
	} else {
		s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// Handler: Get job
func (s *Server) getJob(w http.ResponseWriter, r *http.Request, jobID int) {
	job, err := s.storage.GetJob(r.Context(), jobID)
	if err != nil {
		s.logger.Error("Failed to get job %d: %v", jobID, err)
		s.errorResponse(w, http.StatusNotFound, "Job not found")
		return
	}

	// Get last run
	lastRun, err := s.storage.GetLastJobRun(r.Context(), jobID)
	if err != nil {
		s.logger.Warn("Failed to get last run for job %d: %v", jobID, err)
	}

	response := models.JobWithLastRun{
		Job:     *job,
		LastRun: lastRun,
	}

	s.jsonResponse(w, http.StatusOK, response)
}

// Handler: Update job log
func (s *Server) updateJobLog(w http.ResponseWriter, r *http.Request, jobID int) {
	var req models.UpdateJobRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WorkerID == "" || req.Status == "" {
		s.errorResponse(w, http.StatusBadRequest, "Worker ID and status are required")
		return
	}

	// Get the most recent job run for this job
	lastRun, err := s.storage.GetLastJobRun(r.Context(), jobID)
	if err != nil || lastRun == nil {
		s.logger.Error("Failed to get last job run for job %d: %v", jobID, err)
		s.errorResponse(w, http.StatusNotFound, "Job run not found")
		return
	}

	// Update the job run
	now := time.Now()
	if lastRun.StartTime == nil && req.Status == models.JobStatusRunning {
		lastRun.StartTime = &now
	}
	if req.Status == models.JobStatusSuccess || req.Status == models.JobStatusFailed {
		lastRun.EndTime = &now
	}

	lastRun.WorkerID = req.WorkerID
	lastRun.Status = req.Status
	lastRun.Logs = req.Logs
	lastRun.ExitCode = req.ExitCode

	if err := s.storage.UpdateJobRun(r.Context(), lastRun); err != nil {
		s.logger.Error("Failed to update job run: %v", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to update job run")
		return
	}

	s.logger.Info("Updated job %d status to %s by worker %s", jobID, req.Status, req.WorkerID)
	s.jsonResponse(w, http.StatusOK, lastRun)
}

// Handler: Worker register
func (s *Server) handleWorkerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.RegisterWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WorkerID == "" {
		s.errorResponse(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	worker := &models.Worker{
		WorkerID: req.WorkerID,
	}

	if err := s.storage.RegisterWorker(r.Context(), worker); err != nil {
		s.logger.Error("Failed to register worker %s: %v", req.WorkerID, err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to register worker")
		return
	}

	s.logger.Info("Registered worker: %s", req.WorkerID)
	s.jsonResponse(w, http.StatusOK, worker)
}

// Handler: Worker heartbeat
func (s *Server) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WorkerID == "" {
		s.errorResponse(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	if err := s.storage.UpdateWorkerHeartbeat(r.Context(), req.WorkerID, time.Now()); err != nil {
		s.logger.Error("Failed to update heartbeat for worker %s: %v", req.WorkerID, err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to update heartbeat")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Handler: Worker get next job
func (s *Server) handleWorkerNext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/workers/")
	workerID := parts[0]

	// Get next job for worker from job manager
	if s.jobManager == nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"job": nil})
		return
	}

	job, err := s.jobManager.GetJobForWorker(r.Context(), workerID)
	if err != nil {
		s.logger.Error("Failed to get job for worker %s: %v", workerID, err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to get job")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"job": job})
}}

	workerID := parts[0]

	// This is a placeholder - actual job assignment will be handled by scheduler
	// For now, just return empty to indicate no job available
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"job": nil})
}
