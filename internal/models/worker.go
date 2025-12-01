package models

import "time"

// Worker represents a worker node in the distributed system
type Worker struct {
	WorkerID     string    `json:"worker_id" db:"worker_id"`
	Status       string    `json:"status" db:"status"`
	LastSeen     time.Time `json:"last_seen" db:"last_seen"`
	RegisteredAt time.Time `json:"registered_at" db:"registered_at"`
}

// RegisterWorkerRequest represents the request to register a new worker
type RegisterWorkerRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
}

// HeartbeatRequest represents a worker heartbeat request
type HeartbeatRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
}
