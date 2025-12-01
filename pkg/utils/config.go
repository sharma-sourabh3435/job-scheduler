package utils

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	// Database
	DatabasePath string

	// Scheduler
	SchedulerPort          int
	SchedulerHost          string
	JobAssignmentInterval  time.Duration
	WorkerTimeoutThreshold time.Duration
	HeartbeatCheckInterval time.Duration

	// Worker
	WorkerID                string
	WorkerPollInterval      time.Duration
	WorkerHeartbeatInterval time.Duration
	SchedulerURL            string

	// Logging
	LogLevel string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		// Database
		DatabasePath: getEnv("DB_PATH", "scheduler.db"),

		// Scheduler
		SchedulerPort:          getEnvAsInt("SCHEDULER_PORT", 8080),
		SchedulerHost:          getEnv("SCHEDULER_HOST", "0.0.0.0"),
		JobAssignmentInterval:  getEnvAsDuration("JOB_ASSIGNMENT_INTERVAL", 2*time.Second),
		WorkerTimeoutThreshold: getEnvAsDuration("WORKER_TIMEOUT_THRESHOLD", 30*time.Second),
		HeartbeatCheckInterval: getEnvAsDuration("HEARTBEAT_CHECK_INTERVAL", 10*time.Second),

		// Worker
		WorkerID:                getEnv("WORKER_ID", ""),
		WorkerPollInterval:      getEnvAsDuration("WORKER_POLL_INTERVAL", 5*time.Second),
		WorkerHeartbeatInterval: getEnvAsDuration("WORKER_HEARTBEAT_INTERVAL", 10*time.Second),
		SchedulerURL:            getEnv("SCHEDULER_URL", "http://localhost:8080"),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "INFO"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvAsDuration gets an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetSchedulerAddress returns the full scheduler address
func (c *Config) GetSchedulerAddress() string {
	return c.SchedulerHost + ":" + strconv.Itoa(c.SchedulerPort)
}

// GetLogLevel converts the log level string to LogLevel type
func (c *Config) GetLogLevel() LogLevel {
	switch c.LogLevel {
	case "DEBUG":
		return DEBUG
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}
