package worker

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/sharma-sourabh3435/job-scheduler/pkg/utils"
)

// ExecutionResult holds the result of a job execution
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// Executor executes jobs
type Executor struct {
	logger *utils.Logger
}

// NewExecutor creates a new executor instance
func NewExecutor(logger *utils.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}

// Execute executes a shell command and returns the result
func (e *Executor) Execute(ctx context.Context, command string) ExecutionResult {
	start := time.Now()

	e.logger.Debug("Executing command: %s", command)

	// Create command with shell
	var cmd *exec.Cmd

	// Use PowerShell on Windows, sh on Unix-like systems
	cmd = exec.CommandContext(ctx, "powershell.exe", "-Command", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	duration := time.Since(start)

	result := ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Error:    err,
	}

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		e.logger.Error("Command failed: %v", err)
	} else {
		result.ExitCode = 0
		e.logger.Debug("Command completed successfully in %v", duration)
	}

	return result
}

// ExecuteWithTimeout executes a command with a timeout
func (e *Executor) ExecuteWithTimeout(command string, timeout time.Duration) ExecutionResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return e.Execute(ctx, command)
}
