// Package logger provides structured logging utilities for kubectx-manager.
// It supports different log levels and colored output for better readability.
package logger

import (
	"fmt"
	"os"
)

type Logger struct {
	verbose bool
	quiet   bool
}

func New(verbose, quiet bool) *Logger {
	return &Logger{
		verbose: verbose,
		quiet:   quiet,
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose && !l.quiet {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Printf(format+"\n", args...)
	}
}

func (l *Logger) Warn(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
