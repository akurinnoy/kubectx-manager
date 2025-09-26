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

package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestAskUserAboutConflicts tests the interactive user choice functionality
// Since this function requires user input, we test it by mocking stdin
func TestAskUserAboutConflicts(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		conflicts []string
	}{
		{
			name:      "user chooses no backup",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "n\n",
			expected:  "none",
		},
		{
			name:      "user chooses selective backup",
			conflicts: []string{"context 'prod' (different configuration)", "user 'admin' (different credentials)"},
			input:     "s\n",
			expected:  "selective",
		},
		{
			name:      "user chooses full backup",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "f\n",
			expected:  "full",
		},
		{
			name:      "user chooses cancel",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "c\n",
			expected:  "cancel",
		},
		{
			name:      "user enters 'no' (long form)",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "no\n",
			expected:  "none",
		},
		{
			name:      "user enters 'selective' (long form)",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "selective\n",
			expected:  "selective",
		},
		{
			name:      "user enters 'full' (long form)",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "full\n",
			expected:  "full",
		},
		{
			name:      "user enters 'cancel' (long form)",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "cancel\n",
			expected:  "cancel",
		},
		{
			name:      "user enters invalid choice",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "invalid\n",
			expected:  "cancel",
		},
		{
			name:      "user enters uppercase choice",
			conflicts: []string{"context 'prod' (different configuration)"},
			input:     "S\n",
			expected:  "selective",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original stdin
			oldStdin := os.Stdin

			// Create a pipe to simulate user input
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write the test input to the pipe
			go func() {
				defer w.Close()
				w.WriteString(tt.input)
			}()

			// Capture stdout to avoid printing during tests
			oldStdout := os.Stdout
			r2, w2, _ := os.Pipe()
			os.Stdout = w2

			// Call the function
			result := askUserAboutConflicts(tt.conflicts)

			// Restore original stdout and stdin
			w2.Close()
			os.Stdout = oldStdout
			os.Stdin = oldStdin

			// Read captured output (optional, can be used for verification)
			output := make([]byte, 1024)
			r2.Read(output)
			r2.Close()

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}

			// Verify that conflicts were displayed in output
			outputStr := string(output)
			for _, conflict := range tt.conflicts {
				if !strings.Contains(outputStr, conflict) {
					t.Errorf("Expected output to contain conflict '%s', but it didn't. Output: %s", conflict, outputStr)
				}
			}
		})
	}
}

// TestAskUserAboutConflictsOutput tests that the correct prompts are displayed
func TestAskUserAboutConflictsOutput(t *testing.T) {
	conflicts := []string{
		"context 'production-cluster' (different configuration)",
		"user 'admin-user' (different credentials)",
	}

	// Save original stdin and stdout
	oldStdin := os.Stdin
	oldStdout := os.Stdout

	// Create pipes
	r, w, _ := os.Pipe()
	os.Stdin = r

	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	// Provide input
	go func() {
		defer w.Close()
		w.WriteString("n\n")
	}()

	// Call function
	askUserAboutConflicts(conflicts)

	// Close write end and restore stdout
	w2.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	// Read output
	output := make([]byte, 2048)
	n, _ := r2.Read(output)
	r2.Close()
	outputStr := string(output[:n])

	// Verify expected content is in output
	expectedContent := []string{
		"⚠️  Restoring this backup would overwrite 2 existing items:",
		"- context 'production-cluster' (different configuration)",
		"- user 'admin-user' (different credentials)",
		"Backup options:",
		"1. No backup - proceed anyway (n)",
		"2. Selective backup - backup only conflicting items (s)",
		"3. Full backup - backup entire kubeconfig (f)",
		"4. Cancel restore (c)",
		"Choose (n/s/f/c):",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't. Full output:\n%s", expected, outputStr)
		}
	}
}

// TestConflictDisplayFormatting tests that conflicts are properly formatted
func TestConflictDisplayFormatting(t *testing.T) {
	tests := []struct {
		name              string
		expectedItemCount string
		conflicts         []string
	}{
		{
			name:              "single conflict",
			conflicts:         []string{"context 'prod' (different configuration)"},
			expectedItemCount: "1 existing items",
		},
		{
			name: "multiple conflicts",
			conflicts: []string{
				"context 'prod' (different configuration)",
				"user 'admin' (different credentials)",
				"cluster 'main' (different server/auth)",
			},
			expectedItemCount: "3 existing items",
		},
		{
			name: "five conflicts",
			conflicts: []string{
				"context 'prod1' (different configuration)",
				"context 'prod2' (different configuration)",
				"user 'admin1' (different credentials)",
				"user 'admin2' (different credentials)",
				"cluster 'main' (different server/auth)",
			},
			expectedItemCount: "5 existing items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original stdin and stdout
			oldStdin := os.Stdin
			oldStdout := os.Stdout

			// Create pipes
			r, w, _ := os.Pipe()
			os.Stdin = r

			r2, w2, _ := os.Pipe()
			os.Stdout = w2

			// Provide input
			go func() {
				defer w.Close()
				w.WriteString("c\n") // Cancel to exit quickly
			}()

			// Call function
			askUserAboutConflicts(tt.conflicts)

			// Close write end and restore stdout
			w2.Close()
			os.Stdout = oldStdout
			os.Stdin = oldStdin

			// Read output
			output := make([]byte, 2048)
			n, _ := r2.Read(output)
			r2.Close()
			outputStr := string(output[:n])

			// Verify the item count is correct
			if !strings.Contains(outputStr, tt.expectedItemCount) {
				t.Errorf("Expected output to contain '%s', but it didn't. Full output:\n%s", tt.expectedItemCount, outputStr)
			}

			// Verify each conflict is listed
			for _, conflict := range tt.conflicts {
				if !strings.Contains(outputStr, "- "+conflict) {
					t.Errorf("Expected output to contain '- %s', but it didn't", conflict)
				}
			}
		})
	}
}
