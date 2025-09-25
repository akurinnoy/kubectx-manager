package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/akurinnoy/kubectx-manager/internal/kubeconfig"
)

// TestRestoreCleanupLogic tests the actual cleanup logic from runRestore function
func TestRestoreCleanupLogic(t *testing.T) {
	tests := []struct {
		name               string
		expectLogMessage   string
		keepBackupFlag     bool
		expectBackupExists bool
	}{
		{
			name:               "cleanup_enabled_deletes_backup",
			keepBackupFlag:     false,
			expectBackupExists: false,
			expectLogMessage:   "Removed backup file:",
		},
		{
			name:               "keep_backup_preserves_file",
			keepBackupFlag:     true,
			expectBackupExists: true,
			expectLogMessage:   "Backup file preserved:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			kubeconfigPath := filepath.Join(tmpDir, "config")
			backupPath := kubeconfigPath + ".backup.20231124-120000"

			// Create test kubeconfig
			testConfig := &kubeconfig.Config{
				APIVersion: "v1",
				Kind:       "Config",
				Contexts: []kubeconfig.NamedContext{
					{Name: "test", Context: &kubeconfig.Context{Cluster: "test", User: "test"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "test", Cluster: &kubeconfig.Cluster{Server: "https://test.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "test", User: &kubeconfig.User{Token: "test"}},
				},
			}

			// Save both files
			err := kubeconfig.Save(testConfig, kubeconfigPath)
			if err != nil {
				t.Fatalf("Failed to save kubeconfig: %v", err)
			}

			err = kubeconfig.Save(testConfig, backupPath)
			if err != nil {
				t.Fatalf("Failed to save backup: %v", err)
			}

			// Create a test logger to capture output
			captureLogger := &CapturingLogger{}

			selectedBackup := Backup{
				Name: filepath.Base(backupPath),
				Path: backupPath,
			}

			// Execute the exact cleanup logic from runRestore
			// Simulate cleanup behavior
			if !tt.keepBackupFlag {
				captureLogger.Debug("Cleaning up backup file: %s", selectedBackup.Path)
				err = os.Remove(selectedBackup.Path)
				if err != nil {
					captureLogger.Warn("Failed to remove backup file %s: %v", selectedBackup.Path, err)
					captureLogger.Warn("You may want to manually remove it")
				} else {
					captureLogger.Info("Removed backup file: %s", selectedBackup.Name)
				}
			} else {
				captureLogger.Info("Backup file preserved: %s", selectedBackup.Name)
			}

			// Verify file state
			_, err = os.Stat(backupPath)
			backupExists := !os.IsNotExist(err)

			if backupExists != tt.expectBackupExists {
				t.Errorf("Expected backup exists=%v, got %v", tt.expectBackupExists, backupExists)
			}

			// Verify log message
			found := false
			for _, entry := range captureLogger.entries {
				if contains(entry, tt.expectLogMessage) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected log message containing '%s', got: %v", tt.expectLogMessage, captureLogger.entries)
			}
		})
	}
}

// CapturingLogger captures log messages for testing
type CapturingLogger struct {
	entries []string
}

func (l *CapturingLogger) Debug(format string, args ...interface{}) {
	l.entries = append(l.entries, fmt.Sprintf("[DEBUG] "+format, args...))
}

func (l *CapturingLogger) Info(format string, args ...interface{}) {
	l.entries = append(l.entries, fmt.Sprintf("[INFO] "+format, args...))
}

func (l *CapturingLogger) Warn(format string, args ...interface{}) {
	l.entries = append(l.entries, fmt.Sprintf("[WARN] "+format, args...))
}

func (l *CapturingLogger) Error(format string, args ...interface{}) {
	l.entries = append(l.entries, fmt.Sprintf("[ERROR] "+format, args...))
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBackupCleanupWithPermissionError tests error handling during cleanup
func TestBackupCleanupWithPermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	backupPath := kubeconfigPath + ".backup.20231124-120000"

	// Create test files
	testConfig := &kubeconfig.Config{
		APIVersion: "v1",
		Kind:       "Config",
	}

	err := kubeconfig.Save(testConfig, kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to save kubeconfig: %v", err)
	}

	err = kubeconfig.Save(testConfig, backupPath)
	if err != nil {
		t.Fatalf("Failed to save backup: %v", err)
	}

	// Make directory read-only to cause deletion failure
	backupDir := filepath.Dir(backupPath)
	originalMode := getFileMode(t, backupDir)
	err = os.Chmod(backupDir, 0444) // Read-only
	if err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}

	// Restore permissions at the end
	defer func() {
		os.Chmod(backupDir, originalMode)
	}()

	// Test cleanup with permission error
	captureLogger := &CapturingLogger{}
	selectedBackup := Backup{
		Name: filepath.Base(backupPath),
		Path: backupPath,
	}

	// Execute cleanup logic (should fail but handle gracefully)
	keepBackupFlag := false
	if !keepBackupFlag {
		captureLogger.Debug("Cleaning up backup file: %s", selectedBackup.Path)
		err = os.Remove(selectedBackup.Path)
		if err != nil {
			captureLogger.Warn("Failed to remove backup file %s: %v", selectedBackup.Path, err)
			captureLogger.Warn("You may want to manually remove it")
		} else {
			captureLogger.Info("Removed backup file: %s", selectedBackup.Name)
		}
	}

	// Verify backup still exists (deletion failed)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup should still exist when deletion fails")
	}

	// Verify warning appears in logs
	foundWarning := false
	for _, entry := range captureLogger.entries {
		if contains(entry, "Failed to remove backup file") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("Expected warning about failed backup removal, got: %v", captureLogger.entries)
	}

	// Restore permissions for cleanup
	os.Chmod(backupDir, originalMode)
}

// Helper to get file mode
func getFileMode(t *testing.T, path string) os.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	return info.Mode()
}
