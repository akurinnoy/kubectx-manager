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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFindBackups(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create original file
	err := os.WriteFile(kubeconfigPath, []byte("original content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup files with different timestamps
	backupData := []struct {
		timestamp string
		content   string
	}{
		{"20231201-120000", "backup1"},
		{"20231201-130000", "backup2"},
		{"20231201-140000", "backup3"},
		{"20231202-100000", "backup4"}, // Newer date
	}

	for _, backup := range backupData {
		backupPath := fmt.Sprintf("%s.backup.%s", kubeconfigPath, backup.timestamp)
		err := os.WriteFile(backupPath, []byte(backup.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
	}

	// Create a file that shouldn't be recognized as backup (wrong format)
	wrongBackupPath := kubeconfigPath + ".backup.invalid-format"
	err = os.WriteFile(wrongBackupPath, []byte("invalid"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid backup file: %v", err)
	}

	// Create another file that shouldn't match
	otherPath := filepath.Join(tmpDir, "other.backup.20231201-120000")
	err = os.WriteFile(otherPath, []byte("other"), 0644)
	if err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}

	// Test findBackups function
	backups, err := findBackups(kubeconfigPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should find 4 valid backups
	if len(backups) != 4 {
		t.Errorf("Expected 4 backups, got %d", len(backups))
	}

	// Check that backups are sorted by time (newest first)
	if len(backups) >= 2 {
		if backups[0].Time.Before(backups[1].Time) {
			t.Errorf("Backups are not sorted correctly (newest first)")
		}
	}

	// Check that the newest backup is first
	if len(backups) > 0 {
		expectedNewest := "config.backup.20231202-100000"
		if backups[0].Name != expectedNewest {
			t.Errorf("Expected newest backup to be %s, got %s", expectedNewest, backups[0].Name)
		}
	}

	// Verify backup paths and content
	for _, backup := range backups {
		if !strings.HasPrefix(backup.Name, "config.backup.") {
			t.Errorf("Backup name doesn't have expected prefix: %s", backup.Name)
		}

		if !strings.Contains(backup.Path, tmpDir) {
			t.Errorf("Backup path doesn't contain temp directory: %s", backup.Path)
		}
	}
}

func TestFindBackupsEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Don't create the original file
	backups, err := findBackups(kubeconfigPath)
	if err != nil {
		t.Errorf("Unexpected error for empty directory: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("Expected 0 backups for empty directory, got %d", len(backups))
	}
}

func TestGetUserSelection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxOptions  int
		expected    int
		expectError bool
	}{
		{"valid selection 1", "1\n", 3, 1, false},
		{"valid selection 2", "2\n", 3, 2, false},
		{"valid selection 3", "3\n", 3, 3, false},
		{"cancel with 0", "0\n", 3, 0, false},
		{"out of range high", "4\n1\n", 3, 1, false},          // Should retry and accept 1
		{"out of range low", "-1\n2\n", 3, 2, false},          // Should retry and accept 2
		{"invalid input then valid", "abc\n2\n", 3, 2, false}, // Should retry and accept 2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to capture prompts
			oldStdout := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write input in goroutine
			go func() {
				defer w.Close()
				w.WriteString(tt.input)
			}()

			result, err := getUserSelection(tt.maxOptions)

			// Close and restore
			wOut.Close()
			os.Stdin = oldStdin
			os.Stdout = oldStdout

			// Read output (prompts)
			var output bytes.Buffer
			output.ReadFrom(rOut)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}

			// Verify prompt display
			outputStr := output.String()
			expectedPrompt := fmt.Sprintf("Select backup to restore (1-%d, or 0 to cancel):", tt.maxOptions)
			if !strings.Contains(outputStr, expectedPrompt) {
				t.Errorf("Expected prompt %q in output %q", expectedPrompt, outputStr)
			}
		})
	}
}

func TestConfirmRestore(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes lowercase", "y\n", true},
		{"yes uppercase", "Y\n", true},
		{"yes full", "yes\n", true},
		{"yes full mixed case", "Yes\n", true},
		{"no", "n\n", false},
		{"no uppercase", "N\n", false},
		{"empty input", "\n", false},
		{"other input", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to capture prompt
			oldStdout := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write input
			go func() {
				defer w.Close()
				w.WriteString(tt.input)
			}()

			result := confirmRestore("test.backup.123", "/path/to/config")

			wOut.Close()
			os.Stdin = oldStdin
			os.Stdout = oldStdout

			// Read the prompt output
			var output bytes.Buffer
			output.ReadFrom(rOut)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for input %q", tt.expected, result, tt.input)
			}

			// Verify prompt content
			outputStr := output.String()
			if !strings.Contains(outputStr, "test.backup.123") {
				t.Errorf("Prompt should contain backup name, got: %s", outputStr)
			}
			if !strings.Contains(outputStr, "/path/to/config") {
				t.Errorf("Prompt should contain config path, got: %s", outputStr)
			}
		})
	}
}

func TestRestoreFromBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backup file
	backupPath := filepath.Join(tmpDir, "backup.file")
	backupContent := "backup content"
	err := os.WriteFile(backupPath, []byte(backupContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Create target file with different content
	targetPath := filepath.Join(tmpDir, "target.file")
	originalContent := "original content"
	err = os.WriteFile(targetPath, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Restore from backup
	err = restoreFromBackup(backupPath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify target file has backup content
	restoredContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Errorf("Failed to read restored file: %v", err)
	}

	if string(restoredContent) != backupContent {
		t.Errorf("Expected restored content %q, got %q", backupContent, string(restoredContent))
	}
}

func TestRestoreFromBackupNonExistentBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "nonexistent.backup")
	targetPath := filepath.Join(tmpDir, "target.file")

	err := restoreFromBackup(backupPath, targetPath)
	if err == nil {
		t.Errorf("Expected error for non-existent backup file, got none")
	}
}

func TestBackupTimeFormatParsing(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectValid bool
	}{
		{"valid format", "config.backup.20231201-143022", true},
		{"invalid format - no seconds", "config.backup.20231201-1430", false},
		{"invalid format - wrong separator", "config.backup.20231201_143022", false},
		{"invalid format - wrong date", "config.backup.2023-12-01-143022", false},
		{"invalid format - extra chars", "config.backup.20231201-143022-extra", false},
		{"invalid format - letters", "config.backup.2023120a-143022", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract timestamp from filename
			prefix := "config.backup."
			if !strings.HasPrefix(tt.filename, prefix) {
				if tt.expectValid {
					t.Errorf("Test setup error: filename should start with prefix")
				}
				return
			}

			timestampStr := strings.TrimPrefix(tt.filename, prefix)
			_, err := time.Parse("20060102-150405", timestampStr)

			if tt.expectValid && err != nil {
				t.Errorf("Expected valid timestamp, but got error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Errorf("Expected invalid timestamp, but parsing succeeded")
			}
		})
	}
}

func TestRestoreCommandDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create kubeconfig file
	err := os.WriteFile(kubeconfigPath, []byte("current config"), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Test with no backups
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	var output bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	os.Args = []string{"kubectx-manager", "restore", "--kubeconfig", kubeconfigPath}

	err = Execute()

	w.Close()
	os.Stdout = oldStdout
	output.ReadFrom(r)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "No backups found") {
		t.Errorf("Expected 'No backups found' message, got: %s", outputStr)
	}
}

func TestRestoreWithBackups(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create kubeconfig file
	err := os.WriteFile(kubeconfigPath, []byte("current config"), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Create backup files
	backup1Path := kubeconfigPath + ".backup.20231201-120000"
	backup2Path := kubeconfigPath + ".backup.20231201-130000"

	err = os.WriteFile(backup1Path, []byte("backup1 content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create backup1: %v", err)
	}

	err = os.WriteFile(backup2Path, []byte("backup2 content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create backup2: %v", err)
	}

	// Test finding backups
	backups, err := findBackups(kubeconfigPath)
	if err != nil {
		t.Errorf("Unexpected error finding backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, found %d", len(backups))
	}

	// Verify backup ordering (newest first)
	if len(backups) >= 2 && backups[0].Name != "config.backup.20231201-130000" {
		t.Errorf("Expected newest backup first, got %s", backups[0].Name)
	}
}
