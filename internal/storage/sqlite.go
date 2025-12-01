package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sharma-sourabh3435/job-scheduler/internal/models"
)

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema initializes the database schema
func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		command TEXT NOT NULL,
		schedule TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS job_runs (
		run_id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id INTEGER NOT NULL,
		worker_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		start_time TIMESTAMP,
		end_time TIMESTAMP,
		logs TEXT,
		exit_code INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS workers (
		worker_id TEXT PRIMARY KEY,
		status TEXT NOT NULL DEFAULT 'active',
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_job_runs_job_id ON job_runs(job_id);
	CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs(status);
	CREATE INDEX IF NOT EXISTS idx_job_runs_worker_id ON job_runs(worker_id);
	CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status);
	CREATE INDEX IF NOT EXISTS idx_workers_last_seen ON workers(last_seen);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateJob creates a new job
func (s *SQLiteStorage) CreateJob(ctx context.Context, job *models.Job) error {
	query := `INSERT INTO jobs (name, command, schedule, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := s.db.ExecContext(ctx, query, job.Name, job.Command, job.Schedule, now, now)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	job.ID = int(id)
	job.CreatedAt = now
	job.UpdatedAt = now
	return nil
}

// GetJob retrieves a job by ID
func (s *SQLiteStorage) GetJob(ctx context.Context, id int) (*models.Job, error) {
	query := `SELECT id, name, command, schedule, created_at, updated_at FROM jobs WHERE id = ?`

	job := &models.Job{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.Name, &job.Command, &job.Schedule, &job.CreatedAt, &job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// ListJobs retrieves all jobs with pagination
func (s *SQLiteStorage) ListJobs(ctx context.Context, limit, offset int) ([]*models.Job, error) {
	query := `SELECT id, name, command, schedule, created_at, updated_at 
	          FROM jobs ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		if err := rows.Scan(&job.ID, &job.Name, &job.Command, &job.Schedule, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// UpdateJob updates an existing job
func (s *SQLiteStorage) UpdateJob(ctx context.Context, job *models.Job) error {
	query := `UPDATE jobs SET name = ?, command = ?, schedule = ?, updated_at = ? WHERE id = ?`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, job.Name, job.Command, job.Schedule, now, job.ID)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	job.UpdatedAt = now
	return nil
}

// DeleteJob deletes a job
func (s *SQLiteStorage) DeleteJob(ctx context.Context, id int) error {
	query := `DELETE FROM jobs WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// CreateJobRun creates a new job run
func (s *SQLiteStorage) CreateJobRun(ctx context.Context, jobRun *models.JobRun) error {
	query := `INSERT INTO job_runs (job_id, worker_id, status, start_time, created_at) 
	          VALUES (?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := s.db.ExecContext(ctx, query, jobRun.JobID, jobRun.WorkerID, jobRun.Status, jobRun.StartTime, now)
	if err != nil {
		return fmt.Errorf("failed to create job run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	jobRun.RunID = int(id)
	jobRun.CreatedAt = now
	return nil
}

// GetJobRun retrieves a job run by ID
func (s *SQLiteStorage) GetJobRun(ctx context.Context, runID int) (*models.JobRun, error) {
	query := `SELECT run_id, job_id, worker_id, status, start_time, end_time, logs, exit_code, created_at 
	          FROM job_runs WHERE run_id = ?`

	jobRun := &models.JobRun{}
	err := s.db.QueryRowContext(ctx, query, runID).Scan(
		&jobRun.RunID, &jobRun.JobID, &jobRun.WorkerID, &jobRun.Status,
		&jobRun.StartTime, &jobRun.EndTime, &jobRun.Logs, &jobRun.ExitCode, &jobRun.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job run not found: %d", runID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job run: %w", err)
	}

	return jobRun, nil
}

// UpdateJobRun updates a job run
func (s *SQLiteStorage) UpdateJobRun(ctx context.Context, jobRun *models.JobRun) error {
	query := `UPDATE job_runs SET worker_id = ?, status = ?, start_time = ?, end_time = ?, logs = ?, exit_code = ? 
	          WHERE run_id = ?`

	_, err := s.db.ExecContext(ctx, query,
		jobRun.WorkerID, jobRun.Status, jobRun.StartTime, jobRun.EndTime, jobRun.Logs, jobRun.ExitCode, jobRun.RunID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job run: %w", err)
	}

	return nil
}

// GetJobRunsByJobID retrieves all runs for a specific job
func (s *SQLiteStorage) GetJobRunsByJobID(ctx context.Context, jobID int, limit, offset int) ([]*models.JobRun, error) {
	query := `SELECT run_id, job_id, worker_id, status, start_time, end_time, logs, exit_code, created_at 
	          FROM job_runs WHERE job_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, jobID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get job runs: %w", err)
	}
	defer rows.Close()

	var jobRuns []*models.JobRun
	for rows.Next() {
		jobRun := &models.JobRun{}
		if err := rows.Scan(&jobRun.RunID, &jobRun.JobID, &jobRun.WorkerID, &jobRun.Status,
			&jobRun.StartTime, &jobRun.EndTime, &jobRun.Logs, &jobRun.ExitCode, &jobRun.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan job run: %w", err)
		}
		jobRuns = append(jobRuns, jobRun)
	}

	return jobRuns, rows.Err()
}

// GetLastJobRun retrieves the most recent run for a job
func (s *SQLiteStorage) GetLastJobRun(ctx context.Context, jobID int) (*models.JobRun, error) {
	query := `SELECT run_id, job_id, worker_id, status, start_time, end_time, logs, exit_code, created_at 
	          FROM job_runs WHERE job_id = ? ORDER BY created_at DESC LIMIT 1`

	jobRun := &models.JobRun{}
	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&jobRun.RunID, &jobRun.JobID, &jobRun.WorkerID, &jobRun.Status,
		&jobRun.StartTime, &jobRun.EndTime, &jobRun.Logs, &jobRun.ExitCode, &jobRun.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No runs found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last job run: %w", err)
	}

	return jobRun, nil
}

// GetPendingJobRuns retrieves pending job runs
func (s *SQLiteStorage) GetPendingJobRuns(ctx context.Context, limit int) ([]*models.JobRun, error) {
	query := `SELECT run_id, job_id, worker_id, status, start_time, end_time, logs, exit_code, created_at 
	          FROM job_runs WHERE status = ? ORDER BY created_at ASC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, models.JobStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending job runs: %w", err)
	}
	defer rows.Close()

	var jobRuns []*models.JobRun
	for rows.Next() {
		jobRun := &models.JobRun{}
		if err := rows.Scan(&jobRun.RunID, &jobRun.JobID, &jobRun.WorkerID, &jobRun.Status,
			&jobRun.StartTime, &jobRun.EndTime, &jobRun.Logs, &jobRun.ExitCode, &jobRun.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan job run: %w", err)
		}
		jobRuns = append(jobRuns, jobRun)
	}

	return jobRuns, rows.Err()
}

// GetRunningJobRunsByWorker retrieves running jobs for a specific worker
func (s *SQLiteStorage) GetRunningJobRunsByWorker(ctx context.Context, workerID string) ([]*models.JobRun, error) {
	query := `SELECT run_id, job_id, worker_id, status, start_time, end_time, logs, exit_code, created_at 
	          FROM job_runs WHERE worker_id = ? AND status = ?`

	rows, err := s.db.QueryContext(ctx, query, workerID, models.JobStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("failed to get running job runs: %w", err)
	}
	defer rows.Close()

	var jobRuns []*models.JobRun
	for rows.Next() {
		jobRun := &models.JobRun{}
		if err := rows.Scan(&jobRun.RunID, &jobRun.JobID, &jobRun.WorkerID, &jobRun.Status,
			&jobRun.StartTime, &jobRun.EndTime, &jobRun.Logs, &jobRun.ExitCode, &jobRun.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan job run: %w", err)
		}
		jobRuns = append(jobRuns, jobRun)
	}

	return jobRuns, rows.Err()
}

// RegisterWorker registers a new worker
func (s *SQLiteStorage) RegisterWorker(ctx context.Context, worker *models.Worker) error {
	query := `INSERT INTO workers (worker_id, status, last_seen, registered_at) 
	          VALUES (?, ?, ?, ?) 
	          ON CONFLICT(worker_id) DO UPDATE SET status = ?, last_seen = ?`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		worker.WorkerID, models.WorkerStatusActive, now, now,
		models.WorkerStatusActive, now,
	)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	worker.Status = models.WorkerStatusActive
	worker.LastSeen = now
	worker.RegisteredAt = now
	return nil
}

// GetWorker retrieves a worker by ID
func (s *SQLiteStorage) GetWorker(ctx context.Context, workerID string) (*models.Worker, error) {
	query := `SELECT worker_id, status, last_seen, registered_at FROM workers WHERE worker_id = ?`

	worker := &models.Worker{}
	err := s.db.QueryRowContext(ctx, query, workerID).Scan(
		&worker.WorkerID, &worker.Status, &worker.LastSeen, &worker.RegisteredAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worker not found: %s", workerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	return worker, nil
}

// UpdateWorkerHeartbeat updates the last seen timestamp for a worker
func (s *SQLiteStorage) UpdateWorkerHeartbeat(ctx context.Context, workerID string, lastSeen time.Time) error {
	query := `UPDATE workers SET last_seen = ?, status = ? WHERE worker_id = ?`

	_, err := s.db.ExecContext(ctx, query, lastSeen, models.WorkerStatusActive, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker heartbeat: %w", err)
	}

	return nil
}

// UpdateWorkerStatus updates the status of a worker
func (s *SQLiteStorage) UpdateWorkerStatus(ctx context.Context, workerID string, status string) error {
	query := `UPDATE workers SET status = ? WHERE worker_id = ?`

	_, err := s.db.ExecContext(ctx, query, status, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker status: %w", err)
	}

	return nil
}

// GetActiveWorkers retrieves all active workers
func (s *SQLiteStorage) GetActiveWorkers(ctx context.Context) ([]*models.Worker, error) {
	query := `SELECT worker_id, status, last_seen, registered_at FROM workers WHERE status = ?`

	rows, err := s.db.QueryContext(ctx, query, models.WorkerStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get active workers: %w", err)
	}
	defer rows.Close()

	var workers []*models.Worker
	for rows.Next() {
		worker := &models.Worker{}
		if err := rows.Scan(&worker.WorkerID, &worker.Status, &worker.LastSeen, &worker.RegisteredAt); err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}
		workers = append(workers, worker)
	}

	return workers, rows.Err()
}

// ListWorkers retrieves all workers
func (s *SQLiteStorage) ListWorkers(ctx context.Context) ([]*models.Worker, error) {
	query := `SELECT worker_id, status, last_seen, registered_at FROM workers ORDER BY registered_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}
	defer rows.Close()

	var workers []*models.Worker
	for rows.Next() {
		worker := &models.Worker{}
		if err := rows.Scan(&worker.WorkerID, &worker.Status, &worker.LastSeen, &worker.RegisteredAt); err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}
		workers = append(workers, worker)
	}

	return workers, rows.Err()
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// Ping checks the database connection
func (s *SQLiteStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
