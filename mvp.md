# Distributed Job Scheduler – MVP Design Document # 

## Project Goal: #    
Build a distributed job scheduler that allows users to submit, schedule, and monitor jobs running on multiple worker nodes. The system must support distributed execution, fault tolerance for worker nodes, and be modular enough for future extensions like DAG workflows, leader election, and autoscaling.  

## Tech Stack (MVP): #  

Language: Go  

Database: SQLite or PostgreSQL (shared job metadata)  

Networking: REST (HTTP + JSON)  

Workers: Separate Go processes  

CLI (optional): Go CLI to submit/manage jobs  

Logging: Console + DB logs for simplicity  

## 1. High-Level Architecture ##
+-------------------+        HTTP/gRPC         +-------------------+  
|   Scheduler       |<-----------------------> |      Worker       |  
|  (central process)|                          |  (multiple nodes) |  
+-------------------+        REST API         +-------------------+   
        |  
        | stores metadata  
        v  
+-------------------+  
|   Database        |  
|  (Postgres/SQLite)|  
+-------------------+  

  
## Key Design Decisions for Flexibility: ##

Scheduler and workers communicate over REST → easy to swap for gRPC later.  

Database is abstracted behind an interface → easy to swap SQLite → Postgres/Redis.  

Worker execution is modular → supports shell commands now, can add Python scripts or Docker containers later.  

Heartbeat logic in workers allows scheduler to detect failures → can be extended to multi-scheduler.  

## 2. Components ##
### 2.1 Scheduler ###

Responsibilities:  

Accept new jobs via REST API  

Persist jobs in the database  

Track job status (pending, running, failed, succeeded)  

Assign jobs to available workers  

Track worker heartbeats  

Handle job reassignment if worker dies  

Modules:  

scheduler.go: Job assignment logic  

api.go: REST API endpoints  

worker_manager.go: Tracks active workers and heartbeats  

storage.go: DB interface  

job_runner.go (optional in MVP for testing local execution)  

### 2.2 Worker ###

Responsibilities:  

Register with scheduler  

Poll scheduler for new jobs  

Execute jobs (shell commands for MVP)  

Send logs/status updates back  

Send heartbeat at intervals  

Modules:  

worker.go: Polling & execution loop  

api_client.go: Scheduler API communication  

executor.go: Job execution  

logger.go: Logs to console + DB via scheduler API  

### 2.3 Database ###

Responsibilities:  

Persistent storage for jobs, job runs, worker nodes  

Abstracted so scheduler can support multiple DBs in future  

Tables:  

jobs  
Column	Type	Description  
id	INT PK	Unique job ID  
name	TEXT	Job name  
command	TEXT	Shell command to execute  
schedule	TEXT	Cron string  
created_at	TIMESTAMP	Job creation time  
job_runs  
Column	Type	Description  
run_id	INT PK	Unique run ID  
job_id	INT FK	Linked job  
worker_id	TEXT	Worker executing this run  
start_time	TIMESTAMP	Run start  
end_time	TIMESTAMP	Run end  
status	TEXT	success/failure/running  
logs	TEXT	Job logs  
workers  
Column	Type	Description  
worker_id	TEXT PK	Unique worker identifier  
last_seen	TIMESTAMP	Last heartbeat time  
status	TEXT	active/inactive  
### 3. API Design ###
Scheduler API (REST)  
Method	Endpoint	Description	Payload / Response  
POST	/jobs	Submit new job	{name, command, schedule}  
GET	/jobs/{id}	Get job status & last run	{job info, last run info}  
POST	/workers/register	Worker registration	{worker_id}  
POST	/workers/heartbeat	Worker heartbeat	{worker_id}  
POST	/jobs/{id}/log	Worker submits logs	{worker_id, logs, status}  
GET	/jobs	List all jobs	[job summaries]  
### 4. Worker Protocol ###

Worker starts → POST /workers/register with unique ID  

Worker polls scheduler → GET /workers/{id}/next  

Scheduler returns next job to run or empty  

Worker executes job → POST /jobs/{id}/log with logs + status  

Worker periodically sends heartbeat → POST /workers/heartbeat  

This keeps scheduler aware of worker availability and allows failover for missed jobs.  

### 5. Job Execution Flow ###

Scheduler receives job → saves to jobs table  

Scheduler checks workers for availability  

Scheduler assigns job → updates job_runs table  

Worker executes job → streams logs → updates job_runs status  

Scheduler monitors completion and logs  

If worker fails → scheduler reassigns job to another worker  

This flow is modular, so you can extend:  

Cron scheduling → DAG workflows  

Shell commands → Python scripts, Docker tasks  

Single scheduler → multiple schedulers with leader election  

### 6. Key MVP Design Principles ###

Keep scheduler-worker communication abstracted → can swap REST → gRPC  

Keep database access behind interface → can swap SQLite → Postgres → Redis  

Worker execution modular → different types of jobs later  

Heartbeat/failover ready → distributed architecture foundation  

Minimal feature set now → extend later without breaking API  

### 7. Milestones for MVP ###
Week	Task  
1	Project scaffolding (Go modules, folders) + DB schema + basic CLI  
2	Implement scheduler REST API → add job submission & status endpoints  
3	Implement worker registration, polling, and heartbeat  
4	Implement job assignment logic → workers execute shell commands → update logs/status  
5	Add retry logic for failed jobs + simple job history query  
6	Optional: CLI commands to list jobs, check status, and submit jobs  
### 8. Open Points for Future Extensions ###

Multi-scheduler support → leader election (Raft/etcd)  

Workflow/DAG execution  

Autoscaling workers  

Retry policies with exponential backoff  

Web-based dashboard UI  

Support for Python, Docker, or containerized tasks  

## Summary ##

Your MVP Distributed Job Scheduler:  

Scheduler + multiple workers → distributed execution  

Shared DB → persistent state, failover-ready  

REST API → submit jobs, check status, report logs  

Heartbeat → detect worker failure, ready for reassignment  

Modular design → easily extendable for DAGs, multi-scheduler, autoscaling  

This design keeps the system distributed, modular, and open to change  