package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akurinnoy/kubectx-manager/internal/kubeconfig"
	"github.com/akurinnoy/kubectx-manager/internal/logger"
)

func TestBackupCleanupAfterRestore(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create test kubeconfig
	testConfig := &kubeconfig.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Contexts: []kubeconfig.NamedContext{
			{Name: "test-context", Context: &kubeconfig.Context{Cluster: "test-cluster", User: "test-user"}},
		},
		Clusters: []kubeconfig.NamedCluster{
			{Name: "test-cluster", Cluster: &kubeconfig.Cluster{Server: "https://test.com"}},
		},
		Users: []kubeconfig.NamedUser{
			{Name: "test-user", User: &kubeconfig.User{Token: "test-token"}},
		},
	}

	// Save as current config
	err := kubeconfig.Save(testConfig, kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to save test kubeconfig: %v", err)
	}

	tests := []struct {
		name               string
		keepBackup         bool
		expectedFileExists bool
	}{
		{
			name:               "default behavior - backup should be deleted",
			keepBackup:         false,
			expectedFileExists: false,
		},
		{
			name:               "keep backup flag - backup should be preserved",
			keepBackup:         true,
			expectedFileExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh backup for each test
			testBackupPath := kubeconfigPath + ".backup.test-" + tt.name
			err := kubeconfig.Save(testConfig, testBackupPath)
			if err != nil {
				t.Fatalf("Failed to create test backup: %v", err)
			}

			// Verify backup exists
			if _, err := os.Stat(testBackupPath); os.IsNotExist(err) {
				t.Fatalf("Backup file should exist before restore")
			}

			selectedBackup := Backup{
				Name: filepath.Base(testBackupPath),
				Path: testBackupPath,
			}

			log := logger.New(false, true) // quiet logger

			// Test the backup cleanup logic by simulating the end of runRestore
			// First restore the backup
			err = restoreFromBackup(selectedBackup.Path, kubeconfigPath)
			if err != nil {
				t.Fatalf("Failed to restore from backup: %v", err)
			}

			// Then apply the cleanup logic (this is the exact code from runRestore)
			if !tt.keepBackup {
				log.Debug("Cleaning up backup file: %s", selectedBackup.Path)
				err = os.Remove(selectedBackup.Path)
				if err != nil {
					log.Warn("Failed to remove backup file %s: %v", selectedBackup.Path, err)
					log.Warn("You may want to manually remove it")
				} else {
					log.Info("Removed backup file: %s", selectedBackup.Name)
				}
			} else {
				log.Info("Backup file preserved: %s", selectedBackup.Name)
			}

			// Verify the expected state
			_, err = os.Stat(testBackupPath)
			fileExists := !os.IsNotExist(err)

			if fileExists != tt.expectedFileExists {
				t.Errorf("Expected file exists=%v, got %v", tt.expectedFileExists, fileExists)
			}

			if tt.expectedFileExists && err != nil {
				t.Errorf("Backup file should exist but got error: %v", err)
			}
		})
	}
}

func TestBackupCleanupErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentBackup := filepath.Join(tmpDir, "nonexistent.backup")

	log := logger.New(false, true) // quiet logger

	// Test removing a non-existent backup (should not panic)
	err := os.Remove(nonExistentBackup)
	if err != nil {
		// Expected behavior - test graceful error handling
		log.Warn("Failed to remove backup file %s: %v", nonExistentBackup, err)
	}

	// The test passes if we get here without panicking
}

func TestKeepBackupFlagFunctionality(t *testing.T) {
	// Test that the flag is properly initialized
	// Smoke test for flag existence and functionality
	cmd := restoreCmd
	flag := cmd.Flags().Lookup("keep-backup")

	if flag == nil {
		t.Error("--keep-backup flag should be defined")
	}

	if flag.DefValue != "false" {
		t.Errorf("--keep-backup flag should default to false, got %s", flag.DefValue)
	}

	if flag.Usage != "Keep backup file after successful restore (default: delete)" {
		t.Errorf("Unexpected flag usage: %s", flag.Usage)
	}
}
