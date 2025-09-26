// Package logger provides structured logging utilities for kubectx-manager.
// It supports different log levels and colored output for better readability.
//
// Copyright (c) 2025 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

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

// Debugf outputs debug-level messages when verbose mode is enabled.
// Debug messages are only shown if verbose=true and quiet=false.
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.verbose && !l.quiet {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Infof outputs informational messages unless quiet mode is enabled.
// Info messages are shown unless quiet=true.
func (l *Logger) Infof(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// Warnf outputs warning messages unless quiet mode is enabled.
// Warning messages are shown unless quiet=true.
func (l *Logger) Warnf(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
	}
}

// Errorf outputs error messages that are always shown regardless of quiet mode.
// Error messages cannot be suppressed as they indicate critical issues.
func (l *Logger) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
