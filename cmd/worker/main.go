package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/worker"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

func main() {
	// Parse command-line flags
	var (
		workerID          = flag.String("id", "", "Worker ID (required)")
		schedulerURL      = flag.String("scheduler", "http://localhost:8080", "Scheduler URL")
		pollInterval      = flag.Duration("poll-interval", 5*time.Second, "Job polling interval")
		heartbeatInterval = flag.Duration("heartbeat-interval", 10*time.Second, "Heartbeat interval")
		logLevel          = flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
	)

	flag.Parse()

	// Validate required flags
	if *workerID == "" {
		fmt.Println("Error: Worker ID is required")
		fmt.Println("Usage: worker -id <worker-id> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Set log level
	switch *logLevel {
	case "DEBUG":
		utils.SetDefaultLogLevel(utils.DEBUG)
	case "WARN":
		utils.SetDefaultLogLevel(utils.WARN)
	case "ERROR":
		utils.SetDefaultLogLevel(utils.ERROR)
	default:
		utils.SetDefaultLogLevel(utils.INFO)
	}

	// Create worker configuration
	config := worker.Config{
		WorkerID:          *workerID,
		SchedulerURL:      *schedulerURL,
		PollInterval:      *pollInterval,
		HeartbeatInterval: *heartbeatInterval,
	}

	// Create and start worker
	w := worker.NewWorker(config)

	utils.Info("Starting worker %s", *workerID)
	utils.Info("Scheduler URL: %s", *schedulerURL)
	utils.Info("Poll interval: %v", *pollInterval)
	utils.Info("Heartbeat interval: %v", *heartbeatInterval)

	if err := w.Start(); err != nil {
		utils.Fatal("Worker failed: %v", err)
	}
}
