// Package logger provides structured logging utilities for kubectx-manager.
// It supports different log levels and colored output for better readability.
package logger

import (
	"fmt"
	"os"
)

// Logger provides structured logging with different levels and output control.
// It supports verbose mode for debug output and quiet mode for minimal output.
type Logger struct {
	verbose bool
	quiet   bool
}

// New creates a new Logger instance with the specified settings.
// If verbose is true, debug messages will be shown.
// If quiet is true, only error messages will be shown (quiet overrides verbose).
func New(verbose, quiet bool) *Logger {
	return &Logger{
		verbose: verbose,
		quiet:   quiet,
	}
}

// Debug outputs debug-level messages when verbose mode is enabled.
// Debug messages are only shown if verbose=true and quiet=false.
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose && !l.quiet {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Info outputs informational messages unless quiet mode is enabled.
// Info messages are shown unless quiet=true.
func (l *Logger) Info(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// Warn outputs warning messages unless quiet mode is enabled.
// Warning messages are shown unless quiet=true.
func (l *Logger) Warn(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
