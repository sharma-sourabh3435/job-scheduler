This repo implements a distributed job scheduler.
Currently a very basic version

Note - design docs for MVP version are built using LLM agents 
so I don't get stuck in choice overload. Decision Paralysis is real.
Ideally eventually, I rely less on LLM agents cause focus is on my learning

## Quick Start

### Prerequisites
- Go 1.21 or higher
- GCC (for SQLite driver on Windows, included with MinGW or similar)

### Building

Build the scheduler:
```bash
go build -o bin/scheduler.exe ./cmd/scheduler
```

Build the worker:
```bash
go build -o bin/worker.exe ./cmd/worker
```

### Running the System

**Step 1: Start the Scheduler**

```bash
./bin/scheduler.exe
```

Optional flags:
- `-db`: Database file path (default: `scheduler.db`)
- `-port`: API server port (default: `8080`)
- `-host`: API server host (default: `0.0.0.0`)
- `-log-level`: Log level - DEBUG, INFO, WARN, ERROR (default: `INFO`)

Example:
```bash
./bin/scheduler.exe -port 8080 -log-level DEBUG
```

**Step 2: Start One or More Workers**

In separate terminal windows:

```bash
./bin/worker.exe -id worker-1
```

```bash
./bin/worker.exe -id worker-2
```

Optional flags:
- `-id`: Worker ID (required)
- `-scheduler`: Scheduler URL (default: `http://localhost:8080`)
- `-poll-interval`: Job polling interval (default: `5s`)
- `-heartbeat-interval`: Heartbeat interval (default: `10s`)
- `-log-level`: Log level (default: `INFO`)

Example:
```bash
./bin/worker.exe -id worker-1 -scheduler http://localhost:8080 -poll-interval 3s
```

### Using the API

**Submit a Job**

```bash
curl -X POST http://localhost:8080/jobs -H "Content-Type: application/json" -d "{\"name\":\"hello-job\",\"command\":\"echo 'Hello World'\"}"
```

PowerShell:
```powershell
Invoke-RestMethod -Uri http://localhost:8080/jobs -Method Post -ContentType "application/json" -Body '{"name":"hello-job","command":"echo Hello World"}'
```

**List All Jobs**

```bash
curl http://localhost:8080/jobs
```

PowerShell:
```powershell
Invoke-RestMethod -Uri http://localhost:8080/jobs
```

**Get Job Details**

```bash
curl http://localhost:8080/jobs/1
```

PowerShell:
```powershell
Invoke-RestMethod -Uri http://localhost:8080/jobs/1
```

### Example Workflow

1. Start the scheduler:
   ```bash
   ./bin/scheduler.exe
   ```

2. Start a worker:
   ```bash
   ./bin/worker.exe -id worker-1
   ```

3. Submit a job:
   ```powershell
   Invoke-RestMethod -Uri http://localhost:8080/jobs -Method Post -ContentType "application/json" -Body '{"name":"test-job","command":"echo Testing distributed scheduler"}'
   ```

4. Check job status:
   ```powershell
   Invoke-RestMethod -Uri http://localhost:8080/jobs/1
   ```

The worker will automatically:
- Register with the scheduler
- Poll for available jobs
- Execute the job
- Report results back to the scheduler
- Send periodic heartbeats

### Architecture

```
┌───────────────────┐                      ┌───────────────────┐
│   Scheduler       │ <--- HTTP/gRPC ----> │      Worker       │
│  (central process)│                      │  (multiple nodes) │
└───────────────────┘                      └───────────────────┘
         │
         │ stores metadata
         │ (REST API)
         ↓
┌───────────────────┐
│   Database        │
│  (Postgres/SQLite)│
└───────────────────┘
```

### Key Features

- **Distributed Execution**: Jobs are distributed across multiple worker nodes
- **Fault Tolerance**: Workers send heartbeats; failed workers are detected and jobs are reassigned
- **REST API**: Simple HTTP/JSON interface for job submission and monitoring
- **Persistent Storage**: SQLite database stores job metadata and execution history
- **Graceful Shutdown**: Both scheduler and workers handle shutdown signals cleanly

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/jobs` | Create a new job |
| GET | `/jobs` | List all jobs |
| GET | `/jobs/{id}` | Get job details with last run |
| POST | `/jobs/{id}/log` | Update job status/logs (used by workers) |
| POST | `/workers/register` | Register a worker |
| POST | `/workers/heartbeat` | Send worker heartbeat |
| GET | `/workers/{id}/next` | Poll for next job |

### Troubleshooting

**Workers not receiving jobs:**
- Ensure the scheduler is running
- Check that workers can reach the scheduler URL
- Verify workers are sending heartbeats (check scheduler logs)

**Build errors about SQLite:**
- Install GCC (MinGW on Windows)
- Ensure CGO is enabled: `set CGO_ENABLED=1`

**Jobs stuck in pending:**
- Check if any workers are active
- Verify the scheduler assignment loop is running
- Check scheduler logs for errors