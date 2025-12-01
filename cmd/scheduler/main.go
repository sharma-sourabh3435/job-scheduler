package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/internal/api"
	"github.com/sharma-sourabh3435/job-scheduler/internal/scheduler"
	"github.com/sharma-sourabh3435/job-scheduler/internal/storage"
	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

func main() {
	// Parse command-line flags
	var (
		dbPath   = flag.String("db", "scheduler.db", "Database file path")
		port     = flag.Int("port", 8080, "API server port")
		host     = flag.String("host", "0.0.0.0", "API server host")
		logLevel = flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
	)

	flag.Parse()

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

	utils.Info("Starting Distributed Job Scheduler")
	utils.Info("Database: %s", *dbPath)
	utils.Info("API Server: %s:%d", *host, *port)

	// Initialize storage
	store, err := storage.NewSQLiteStorage(*dbPath)
	if err != nil {
		utils.Fatal("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	utils.Info("Database initialized successfully")

	// Create scheduler
	schedulerConfig := scheduler.Config{
		Storage:            store,
		AssignmentInterval: 2 * time.Second,
		WorkerTimeout:      30 * time.Second,
	}

	sched := scheduler.NewScheduler(schedulerConfig)

	// Start scheduler
	if err := sched.Start(); err != nil {
		utils.Fatal("Failed to start scheduler: %v", err)
	}

	// Create API server
	addr := fmt.Sprintf("%s:%d", *host, *port)
	apiServer := api.NewServer(store, sched.GetJobManager(), addr)

	// Start API server in goroutine
	go func() {
		if err := apiServer.Start(); err != nil {
			utils.Error("API server error: %v", err)
		}
	}()

	utils.Info("Scheduler started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	utils.Info("Received shutdown signal")

	// Graceful shutdown
	utils.Info("Shutting down gracefully...")

	// Shutdown API server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		utils.Error("API server shutdown error: %v", err)
	}

	// Stop scheduler
	sched.Stop()

	utils.Info("Shutdown complete")
}
