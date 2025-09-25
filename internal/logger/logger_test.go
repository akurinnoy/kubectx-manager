package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		quiet   bool
	}{
		{"default", false, false},
		{"verbose", true, false},
		{"quiet", false, true},
		{"verbose and quiet", true, true}, // quiet should override verbose
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.verbose, tt.quiet)
			if logger.verbose != tt.verbose {
				t.Errorf("Expected verbose=%v, got %v", tt.verbose, logger.verbose)
			}
			if logger.quiet != tt.quiet {
				t.Errorf("Expected quiet=%v, got %v", tt.quiet, logger.quiet)
			}
		})
	}
}

func TestDebug(t *testing.T) {
	tests := []struct {
		name           string
		expectedPrefix string
		verbose        bool
		quiet          bool
		expectOutput   bool
	}{
		{"verbose mode", "[DEBUG]", true, false, true},
		{"normal mode", "", false, false, false},
		{"quiet mode", "", false, true, false},
		{"verbose + quiet", "", true, true, false}, // quiet overrides verbose
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			logger := New(tt.verbose, tt.quiet)
			logger.Debug("test message %s", "arg")

			w.Close()
			os.Stderr = oldStderr

			var output bytes.Buffer
			output.ReadFrom(r)
			outputStr := output.String()

			if tt.expectOutput {
				if outputStr == "" {
					t.Errorf("Expected output, got none")
				}
				if !strings.Contains(outputStr, tt.expectedPrefix) {
					t.Errorf("Expected prefix %q, got %q", tt.expectedPrefix, outputStr)
				}
				if !strings.Contains(outputStr, "test message arg") {
					t.Errorf("Expected formatted message, got %q", outputStr)
				}
			} else if outputStr != "" {
				t.Errorf("Expected no output, got %q", outputStr)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		quiet        bool
		expectOutput bool
	}{
		{"verbose mode", true, false, true},
		{"normal mode", false, false, true},
		{"quiet mode", false, true, false},
		{"verbose + quiet", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := New(tt.verbose, tt.quiet)
			logger.Info("test info %s", "message")

			w.Close()
			os.Stdout = oldStdout

			var output bytes.Buffer
			output.ReadFrom(r)
			outputStr := output.String()

			if tt.expectOutput {
				if outputStr == "" {
					t.Errorf("Expected output, got none")
				}
				if !strings.Contains(outputStr, "test info message") {
					t.Errorf("Expected formatted message, got %q", outputStr)
				}
			} else if outputStr != "" {
				t.Errorf("Expected no output, got %q", outputStr)
			}
		})
	}
}

func TestWarn(t *testing.T) {
	tests := []struct {
		name           string
		expectedPrefix string
		verbose        bool
		quiet          bool
		expectOutput   bool
	}{
		{"verbose mode", "[WARN]", true, false, true},
		{"normal mode", "[WARN]", false, false, true},
		{"quiet mode", "", false, true, false},
		{"verbose + quiet", "", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			logger := New(tt.verbose, tt.quiet)
			logger.Warn("test warning %s", "message")

			w.Close()
			os.Stderr = oldStderr

			var output bytes.Buffer
			output.ReadFrom(r)
			outputStr := output.String()

			if tt.expectOutput {
				if outputStr == "" {
					t.Errorf("Expected output, got none")
				}
				if !strings.Contains(outputStr, tt.expectedPrefix) {
					t.Errorf("Expected prefix %q, got %q", tt.expectedPrefix, outputStr)
				}
				if !strings.Contains(outputStr, "test warning message") {
					t.Errorf("Expected formatted message, got %q", outputStr)
				}
			} else if outputStr != "" {
				t.Errorf("Expected no output, got %q", outputStr)
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name           string
		expectedPrefix string
		verbose        bool
		quiet          bool
		expectOutput   bool
	}{
		{"verbose mode", "[ERROR]", true, false, true},
		{"normal mode", "[ERROR]", false, false, true},
		{"quiet mode", "[ERROR]", false, true, true}, // Errors always show
		{"verbose + quiet", "[ERROR]", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			logger := New(tt.verbose, tt.quiet)
			logger.Error("test error %s", "message")

			w.Close()
			os.Stderr = oldStderr

			var output bytes.Buffer
			output.ReadFrom(r)
			outputStr := output.String()

			if tt.expectOutput {
				if outputStr == "" {
					t.Errorf("Expected output, got none")
				}
				if !strings.Contains(outputStr, tt.expectedPrefix) {
					t.Errorf("Expected prefix %q, got %q", tt.expectedPrefix, outputStr)
				}
				if !strings.Contains(outputStr, "test error message") {
					t.Errorf("Expected formatted message, got %q", outputStr)
				}
			} else if outputStr != "" {
				t.Errorf("Expected no output, got %q", outputStr)
			}
		})
	}
}

func TestLoggerBehaviorMatrix(t *testing.T) {
	// Test all combinations of verbose/quiet with all log levels
	combinations := []struct {
		level   string
		verbose bool
		quiet   bool
		expect  bool
	}{
		// Normal mode (verbose=false, quiet=false)
		{"debug", false, false, false},
		{"info", false, false, true},
		{"warn", false, false, true},
		{"error", false, false, true},

		// Verbose mode (verbose=true, quiet=false)
		{"debug", true, false, true},
		{"info", true, false, true},
		{"warn", true, false, true},
		{"error", true, false, true},

		// Quiet mode (verbose=false, quiet=true)
		{"debug", false, true, false},
		{"info", false, true, false},
		{"warn", false, true, false},
		{"error", false, true, true}, // Errors always show

		// Verbose + Quiet (quiet overrides)
		{"debug", true, true, false},
		{"info", true, true, false},
		{"warn", true, true, false},
		{"error", true, true, true}, // Errors always show
	}

	for _, combo := range combinations {
		t.Run(testName(combo.verbose, combo.quiet, combo.level), func(t *testing.T) {
			logger := New(combo.verbose, combo.quiet)

			// Capture both stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr

			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()

			os.Stdout = wOut
			os.Stderr = wErr

			// Call the appropriate log method
			switch combo.level {
			case "debug":
				logger.Debug("test")
			case "info":
				logger.Info("test")
			case "warn":
				logger.Warn("test")
			case "error":
				logger.Error("test")
			}

			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			var outputOut, outputErr bytes.Buffer
			outputOut.ReadFrom(rOut)
			outputErr.ReadFrom(rErr)

			totalOutput := outputOut.String() + outputErr.String()
			hasOutput := totalOutput != ""

			if hasOutput != combo.expect {
				t.Errorf("Expected output=%v, got output=%v (content: %q)",
					combo.expect, hasOutput, totalOutput)
			}
		})
	}
}

func testName(verbose, quiet bool, level string) string {
	var mode string
	if verbose && quiet {
		mode = "verbose+quiet"
	} else if verbose {
		mode = "verbose"
	} else if quiet {
		mode = "quiet"
	} else {
		mode = "normal"
	}
	return mode + "_" + level
}
