package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

// Logger provides structured logging capabilities
type Logger struct {
	logger    *log.Logger
	minLevel  LogLevel
	component string
}

// NewLogger creates a new logger instance
func NewLogger(component string, minLevel LogLevel) *Logger {
	return &Logger{
		logger:    log.New(os.Stdout, "", 0),
		minLevel:  minLevel,
		component: component,
	}
}

// shouldLog checks if a message at the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
	}
	return levels[level] >= levels[l.minLevel]
}

// formatMessage formats a log message with timestamp, level, and component
func (l *Logger) formatMessage(level LogLevel, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, level, l.component, message)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, args ...interface{}) {
	if l.shouldLog(DEBUG) {
		l.logger.Println(l.formatMessage(DEBUG, fmt.Sprintf(message, args...)))
	}
}

// Info logs an info message
func (l *Logger) Info(message string, args ...interface{}) {
	if l.shouldLog(INFO) {
		l.logger.Println(l.formatMessage(INFO, fmt.Sprintf(message, args...)))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(message string, args ...interface{}) {
	if l.shouldLog(WARN) {
		l.logger.Println(l.formatMessage(WARN, fmt.Sprintf(message, args...)))
	}
}

// Error logs an error message
func (l *Logger) Error(message string, args ...interface{}) {
	if l.shouldLog(ERROR) {
		l.logger.Println(l.formatMessage(ERROR, fmt.Sprintf(message, args...)))
	}
}

// Fatal logs an error message and exits the program
func (l *Logger) Fatal(message string, args ...interface{}) {
	l.logger.Fatalln(l.formatMessage(ERROR, fmt.Sprintf(message, args...)))
}

// WithComponent creates a new logger with a different component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		logger:    l.logger,
		minLevel:  l.minLevel,
		component: component,
	}
}

// Global logger instance for convenience
var defaultLogger = NewLogger("app", INFO)

// SetDefaultLogLevel sets the log level for the default logger
func SetDefaultLogLevel(level LogLevel) {
	defaultLogger.minLevel = level
}

// Debug logs a debug message using the default logger
func Debug(message string, args ...interface{}) {
	defaultLogger.Debug(message, args...)
}

// Info logs an info message using the default logger
func Info(message string, args ...interface{}) {
	defaultLogger.Info(message, args...)
}

// Warn logs a warning message using the default logger
func Warn(message string, args ...interface{}) {
	defaultLogger.Warn(message, args...)
}

// Error logs an error message using the default logger
func Error(message string, args ...interface{}) {
	defaultLogger.Error(message, args...)
}

// Fatal logs an error message and exits using the default logger
func Fatal(message string, args ...interface{}) {
	defaultLogger.Fatal(message, args...)
}
