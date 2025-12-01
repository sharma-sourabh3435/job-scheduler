# Implementation TODO List - Distributed Job Scheduler

## Phase 1: Project Setup & Foundation (Week 1)

- [ ] **1.1 Initialize Go Project**
  - [ ] Run `go mod init github.com/sharma-sourabh3435/job-scheduler`
  - [ ] Create folder structure as per project-structure.md
  - [ ] Set up .gitignore for Go projects

- [ ] **1.2 Database Schema & Models**
  - [ ] Create `scripts/init_db.sql` with tables (jobs, job_runs, workers)
  - [ ] Define `internal/models/job.go` struct
  - [ ] Define `internal/models/job_run.go` struct
  - [ ] Define `internal/models/worker.go` struct
  - [ ] Add appropriate struct tags for JSON and DB mapping

- [ ] **1.3 Database Layer**
  - [ ] Create `internal/storage/storage.go` interface
    - Methods: CreateJob, GetJob, ListJobs, UpdateJobStatus
    - Methods: CreateJobRun, UpdateJobRun, GetJobRun
    - Methods: RegisterWorker, UpdateWorkerHeartbeat, GetActiveWorkers
  - [ ] Implement SQLite adapter (start simple)
  - [ ] Add database connection pooling
  - [ ] Write basic unit tests for storage layer

- [ ] **1.4 Utilities & Logging**
  - [ ] Create `pkg/utils/logger.go` with structured logging
  - [ ] Add environment variable configuration support
  - [ ] Create constants file for status values (pending, running, success, failed)

## Phase 2: Scheduler REST API (Week 2)

- [ ] **2.1 API Server Setup**
  - [ ] Choose HTTP router (net/http, chi, gin, or gorilla/mux)
  - [ ] Create `internal/api/api.go` with server initialization
  - [ ] Add middleware: logging, CORS, error handling
  - [ ] Implement graceful shutdown

- [ ] **2.2 Job Management Endpoints**
  - [ ] `POST /jobs` - Create new job
    - Parse request body (name, command, schedule)
    - Validate input
    - Store in database
    - Return job ID and status
  - [ ] `GET /jobs` - List all jobs
    - Add pagination support
    - Return job summaries
  - [ ] `GET /jobs/{id}` - Get job details
    - Return job info + last run info
    - Handle job not found

- [ ] **2.3 Worker Management Endpoints**
  - [ ] `POST /workers/register` - Register worker
    - Generate or accept worker_id
    - Store in workers table with status "active"
    - Return registration confirmation
  - [ ] `POST /workers/heartbeat` - Worker heartbeat
    - Update last_seen timestamp
    - Update worker status to active
  - [ ] `GET /workers/{id}/next` - Poll for next job
    - Return pending job or empty response
    - Mark job as assigned to this worker

- [ ] **2.4 Job Logging Endpoint**
  - [ ] `POST /jobs/{id}/log` - Submit job logs and status
    - Update job_run with logs
    - Update status (running/success/failed)
    - Update end_time if completed

- [ ] **2.5 API Testing**
  - [ ] Write integration tests for all endpoints
  - [ ] Test error cases (invalid input, missing jobs, etc.)

## Phase 3: Worker Implementation (Week 3)

- [ ] **3.1 Worker Core**
  - [ ] Create `internal/worker/worker.go`
  - [ ] Implement worker initialization with unique ID
  - [ ] Create configuration struct (scheduler URL, poll interval, heartbeat interval)

- [ ] **3.2 Worker Registration & Heartbeat**
  - [ ] Implement registration on startup
    - POST to /workers/register
    - Handle registration failures with retry
  - [ ] Implement heartbeat goroutine
    - Send heartbeat every N seconds
    - Handle scheduler unavailability

- [ ] **3.3 Job Polling**
  - [ ] Implement polling loop
    - GET /workers/{id}/next at regular intervals
    - Parse job response
    - Handle no-job-available gracefully
  - [ ] Add backoff strategy for failed polls

- [ ] **3.4 Job Executor**
  - [ ] Create `internal/worker/executor.go`
  - [ ] Implement shell command execution
    - Use exec.Command for shell commands
    - Capture stdout and stderr
    - Handle timeouts
    - Capture exit codes
  - [ ] Stream logs back to scheduler
  - [ ] Update job status (running → success/failed)

- [ ] **3.5 Worker Entry Point**
  - [ ] Create `cmd/worker/main.go`
  - [ ] Add command-line flags (scheduler-url, worker-id)
  - [ ] Implement graceful shutdown (complete current job before exit)

- [ ] **3.6 Worker Testing**
  - [ ] Unit tests for executor
  - [ ] Integration tests with mock scheduler

## Phase 4: Scheduler Job Assignment Logic (Week 4)

- [ ] **4.1 Job Manager**
  - [ ] Create `internal/scheduler/job_manager.go`
  - [ ] Implement job queue (in-memory for MVP)
  - [ ] Add job to queue when created via API
  - [ ] Implement GetNextJob logic (FIFO for MVP)

- [ ] **4.2 Worker Manager**
  - [ ] Create `internal/scheduler/worker_manager.go`
  - [ ] Track active workers in memory
  - [ ] Implement heartbeat monitoring
    - Background goroutine to check worker last_seen
    - Mark workers as inactive if no heartbeat for X seconds
  - [ ] Implement GetAvailableWorker logic

- [ ] **4.3 Core Scheduler**
  - [ ] Create `internal/scheduler/scheduler.go`
  - [ ] Implement job assignment loop
    - Check for pending jobs
    - Find available worker
    - Assign job to worker (create job_run record)
    - Update job status to "running"
  - [ ] Run assignment loop in background goroutine

- [ ] **4.4 Scheduler Entry Point**
  - [ ] Create `cmd/scheduler/main.go`
  - [ ] Initialize database
  - [ ] Start API server
  - [ ] Start job assignment loop
  - [ ] Start worker heartbeat monitor
  - [ ] Implement graceful shutdown

- [ ] **4.5 End-to-End Testing**
  - [ ] Test full flow: submit job → assign → execute → log → complete
  - [ ] Test with multiple workers
  - [ ] Verify job_runs table is populated correctly

## Phase 5: Fault Tolerance & Retry Logic (Week 5)

- [ ] **5.1 Job Reassignment on Worker Failure**
  - [ ] Detect when worker becomes inactive with running job
  - [ ] Mark job_run as failed
  - [ ] Requeue job for reassignment
  - [ ] Test worker crash scenarios

- [ ] **5.2 Retry Logic**
  - [ ] Add retry_count field to jobs table
  - [ ] Add max_retries field to jobs table
  - [ ] Implement retry logic in scheduler
    - Check retry_count < max_retries
    - Requeue failed jobs
    - Increment retry_count
  - [ ] Add exponential backoff (optional enhancement)

- [ ] **5.3 Job History & Cleanup**
  - [ ] Create endpoint `GET /jobs/{id}/runs` - List all runs for a job
  - [ ] Add filtering by status (success/failed)
  - [ ] Implement log rotation or cleanup for old job_runs

- [ ] **5.4 Enhanced Error Handling**
  - [ ] Better error messages in API responses
  - [ ] Structured error logging
  - [ ] Dead letter queue for permanently failed jobs (optional)

## Phase 6: CLI & Documentation (Week 6)

- [ ] **6.1 CLI Tool (Optional)**
  - [ ] Create `cmd/cli/main.go`
  - [ ] Implement commands:
    - `job submit --name "..." --command "..."`
    - `job list`
    - `job status <id>`
    - `job logs <id>`
    - `worker list`
  - [ ] Use a CLI framework (cobra or urfave/cli)

- [ ] **6.2 Documentation**
  - [ ] Update README.md
    - Architecture overview
    - Getting started guide
    - API documentation
    - Configuration options
  - [ ] Add code comments and godoc
  - [ ] Create CONTRIBUTING.md
  - [ ] Add example job submissions

- [ ] **6.3 Testing & Quality**
  - [ ] Achieve >80% code coverage
  - [ ] Add golangci-lint configuration
  - [ ] Fix all linting issues
  - [ ] Performance testing with concurrent workers

- [ ] **6.4 Deployment**
  - [ ] Create Docker files for scheduler and worker
  - [ ] Add docker-compose.yml for local testing
  - [ ] Create deployment guide
  - [ ] Add health check endpoints

## Phase 7: Future Enhancements (Post-MVP)

- [ ] **7.1 Advanced Scheduling**
  - [ ] Implement cron scheduling
  - [ ] Add scheduled job execution
  - [ ] Support for recurring jobs

- [ ] **7.2 PostgreSQL Support**
  - [ ] Implement PostgreSQL adapter
  - [ [ ] Add database migration tool (golang-migrate)
  - [ ] Support connection strings in config

- [ ] **7.3 Web Dashboard**
  - [ ] Design UI mockups
  - [ ] Implement frontend (React/Vue)
  - [ ] Real-time job status updates (WebSocket)

- [ ] **7.4 Multi-Scheduler Support**
  - [ ] Implement leader election (Raft/etcd)
  - [ ] Add scheduler failover
  - [ ] Distributed state management

- [ ] **7.5 DAG Workflows**
  - [ ] Design DAG data model
  - [ ] Implement dependency resolution
  - [ ] Add workflow visualization

- [ ] **7.6 Container Support**
  - [ ] Docker executor for containerized jobs
  - [ ] Kubernetes integration
  - [ ] Resource limits and quotas

---

## Quick Start Checklist

When beginning implementation, complete in this order:

1. ✅ Project setup (go.mod, folders)
2. ✅ Database schema + models
3. ✅ Storage interface + SQLite implementation
4. ✅ Basic API server with job endpoints
5. ✅ Worker with registration and execution
6. ✅ Scheduler assignment logic
7. ✅ Test end-to-end flow
8. ✅ Add fault tolerance
9. ✅ Polish and document
