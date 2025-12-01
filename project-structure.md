```
distributed-job-scheduler/
├── cmd/
│   ├── scheduler/
│   │   └── main.go          # Entry point for the scheduler service
│   └── worker/
│       └── main.go          # Entry point for worker node
├── internal/
│   ├── scheduler/
│   │   ├── scheduler.go     # Core scheduling logic
│   │   ├── job_manager.go   # Job assignment and queueing
│   │   └── worker_manager.go# Track workers and heartbeats
│   ├── worker/
│   │   ├── worker.go        # Worker polling loop and execution
│   │   └── executor.go      # Execute jobs (shell command, future extensions)
│   ├── api/
│   │   └── api.go           # REST API handlers (jobs, workers)
│   ├── storage/
│   │   └── storage.go       # DB interface (SQLite/Postgres)
│   └── models/
│       ├── job.go           # Job struct
│       ├── job_run.go       # Job run struct
│       └── worker.go        # Worker struct
├── pkg/
│   └── utils/
│       └── logger.go        # Logging utilities
├── scripts/
│   └── init_db.sql          # Optional DB initialization script
├── go.mod
└── README.md
```