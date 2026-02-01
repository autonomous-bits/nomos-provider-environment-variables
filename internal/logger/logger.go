// Package logger provides structured logging functionality for the provider.
package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Level represents the logging level
type Level int

const (
	// ERROR represents error-level logging.
	ERROR Level = iota
	// WARN represents warning-level logging.
	WARN
	// INFO represents info-level logging.
	INFO
	// DEBUG represents debug-level logging.
	DEBUG
)

func (l Level) String() string {
	switch l {
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging to stderr
type Logger struct {
	level  Level
	output io.Writer
}

// New creates a new logger with the specified minimum level
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		output: os.Stderr,
	}
}

// NewWithOutput creates a logger with custom output (for testing)
func NewWithOutput(level Level, output io.Writer) *Logger {
	return &Logger{
		level:  level,
		output: output,
	}
}

// log writes a log message at the specified level
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level > l.level {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(l.output, "[%s] %s: %s\n", timestamp, level.String(), message)
}

// Error logs an error-level message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Warn logs a warning-level message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Info logs an info-level message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Debug logs a debug-level message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}
