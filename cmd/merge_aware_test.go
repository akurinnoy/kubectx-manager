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
	"path/filepath"
	"strings"
	"testing"

	"github.com/che-incubator/kubectx-manager/internal/kubeconfig"
	"github.com/che-incubator/kubectx-manager/internal/logger"
)

func TestAnalyzeRestoreConflicts(t *testing.T) {
	tests := []struct {
		name              string
		currentConfig     *kubeconfig.Config
		backupConfig      *kubeconfig.Config
		expectedConflicts []string
	}{
		{
			name: "no conflicts - completely different contexts",
			currentConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "current-ctx", Context: &kubeconfig.Context{Cluster: "current-cluster", User: "current-user"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "current-cluster", Cluster: &kubeconfig.Cluster{Server: "https://current.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "current-user", User: &kubeconfig.User{Token: "current-token"}},
				},
			},
			backupConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "backup-ctx", Context: &kubeconfig.Context{Cluster: "backup-cluster", User: "backup-user"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "backup-cluster", Cluster: &kubeconfig.Cluster{Server: "https://backup.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "backup-user", User: &kubeconfig.User{Token: "backup-token"}},
				},
			},
			expectedConflicts: []string{},
		},
		{
			name: "context conflict - same name different config",
			currentConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "prod-ctx", Context: &kubeconfig.Context{Cluster: "cluster-a", User: "user-a"}},
				},
			},
			backupConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "prod-ctx", Context: &kubeconfig.Context{Cluster: "cluster-b", User: "user-b"}},
				},
			},
			expectedConflicts: []string{"context 'prod-ctx' (different configuration)"},
		},
		{
			name: "cluster conflict - same name different server",
			currentConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "prod-cluster", Cluster: &kubeconfig.Cluster{Server: "https://old.com"}},
				},
			},
			backupConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "prod-cluster", Cluster: &kubeconfig.Cluster{Server: "https://new.com"}},
				},
			},
			expectedConflicts: []string{"cluster 'prod-cluster' (different server/auth)"},
		},
		{
			name: "user conflict - same name different credentials",
			currentConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{},
				Users: []kubeconfig.NamedUser{
					{Name: "admin", User: &kubeconfig.User{Token: "old-token"}},
				},
			},
			backupConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{},
				Users: []kubeconfig.NamedUser{
					{Name: "admin", User: &kubeconfig.User{Token: "new-token"}},
				},
			},
			expectedConflicts: []string{"user 'admin' (different credentials)"},
		},
		{
			name: "multiple conflicts",
			currentConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "ctx1", Context: &kubeconfig.Context{Cluster: "cluster-old", User: "user1"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "shared-cluster", Cluster: &kubeconfig.Cluster{Server: "https://old.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "shared-user", User: &kubeconfig.User{Token: "old-token"}},
				},
			},
			backupConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "ctx1", Context: &kubeconfig.Context{Cluster: "cluster-new", User: "user1"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "shared-cluster", Cluster: &kubeconfig.Cluster{Server: "https://new.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "shared-user", User: &kubeconfig.User{Token: "new-token"}},
				},
			},
			expectedConflicts: []string{
				"context 'ctx1' (different configuration)",
				"cluster 'shared-cluster' (different server/auth)",
				"user 'shared-user' (different credentials)",
			},
		},
		{
			name: "identical configs - no conflicts",
			currentConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "ctx1", Context: &kubeconfig.Context{Cluster: "cluster1", User: "user1"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "cluster1", Cluster: &kubeconfig.Cluster{Server: "https://same.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "user1", User: &kubeconfig.User{Token: "same-token"}},
				},
			},
			backupConfig: &kubeconfig.Config{
				Contexts: []kubeconfig.NamedContext{
					{Name: "ctx1", Context: &kubeconfig.Context{Cluster: "cluster1", User: "user1"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "cluster1", Cluster: &kubeconfig.Cluster{Server: "https://same.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "user1", User: &kubeconfig.User{Token: "same-token"}},
				},
			},
			expectedConflicts: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save configs to temporary files and load them properly (to build internal maps)
			tmpDir := t.TempDir()
			currentPath := filepath.Join(tmpDir, "current")
			backupPath := filepath.Join(tmpDir, "backup")

			err := kubeconfig.Save(tt.currentConfig, currentPath)
			if err != nil {
				t.Fatalf("Failed to save current config: %v", err)
			}

			err = kubeconfig.Save(tt.backupConfig, backupPath)
			if err != nil {
				t.Fatalf("Failed to save backup config: %v", err)
			}

			// Load them back (this builds internal maps)
			currentConfig, err := kubeconfig.Load(currentPath)
			if err != nil {
				t.Fatalf("Failed to load current config: %v", err)
			}

			backupConfig, err := kubeconfig.Load(backupPath)
			if err != nil {
				t.Fatalf("Failed to load backup config: %v", err)
			}

			log := logger.New(false, true) // quiet logger for tests
			conflicts := analyzeRestoreConflicts(currentConfig, backupConfig, log)

			if len(conflicts) != len(tt.expectedConflicts) {
				t.Errorf("Expected %d conflicts, got %d: %v", len(tt.expectedConflicts), len(conflicts), conflicts)
				return
			}

			// Check each expected conflict is present
			for _, expected := range tt.expectedConflicts {
				found := false
				for _, actual := range conflicts {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected conflict '%s' not found in %v", expected, conflicts)
				}
			}
		})
	}
}

func TestContextsEqual(t *testing.T) {
	tests := []struct {
		a        *kubeconfig.Context
		b        *kubeconfig.Context
		name     string
		expected bool
	}{
		{
			name:     "identical contexts",
			a:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "ns1"},
			b:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "ns1"},
			expected: true,
		},
		{
			name:     "different cluster",
			a:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "ns1"},
			b:        &kubeconfig.Context{Cluster: "c2", User: "u1", Namespace: "ns1"},
			expected: false,
		},
		{
			name:     "different user",
			a:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "ns1"},
			b:        &kubeconfig.Context{Cluster: "c1", User: "u2", Namespace: "ns1"},
			expected: false,
		},
		{
			name:     "different namespace",
			a:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "ns1"},
			b:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "ns2"},
			expected: false,
		},
		{
			name:     "empty namespace vs set namespace",
			a:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: ""},
			b:        &kubeconfig.Context{Cluster: "c1", User: "u1", Namespace: "default"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contextsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClustersEqual(t *testing.T) {
	tests := []struct {
		a        *kubeconfig.Cluster
		b        *kubeconfig.Cluster
		name     string
		expected bool
	}{
		{
			name: "identical clusters",
			a: &kubeconfig.Cluster{
				Server:                   "https://api.example.com",
				CertificateAuthorityData: "cert-data",
				InsecureSkipTLSVerify:    false,
			},
			b: &kubeconfig.Cluster{
				Server:                   "https://api.example.com",
				CertificateAuthorityData: "cert-data",
				InsecureSkipTLSVerify:    false,
			},
			expected: true,
		},
		{
			name: "different server",
			a: &kubeconfig.Cluster{
				Server: "https://api1.example.com",
			},
			b: &kubeconfig.Cluster{
				Server: "https://api2.example.com",
			},
			expected: false,
		},
		{
			name: "different certificate data",
			a: &kubeconfig.Cluster{
				Server:                   "https://api.example.com",
				CertificateAuthorityData: "cert-data-1",
			},
			b: &kubeconfig.Cluster{
				Server:                   "https://api.example.com",
				CertificateAuthorityData: "cert-data-2",
			},
			expected: false,
		},
		{
			name: "different insecure skip TLS",
			a: &kubeconfig.Cluster{
				Server:                "https://api.example.com",
				InsecureSkipTLSVerify: true,
			},
			b: &kubeconfig.Cluster{
				Server:                "https://api.example.com",
				InsecureSkipTLSVerify: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clustersEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUsersEqual(t *testing.T) {
	tests := []struct {
		a        *kubeconfig.User
		b        *kubeconfig.User
		name     string
		expected bool
	}{
		{
			name: "identical token users",
			a: &kubeconfig.User{
				Token: "abc123",
			},
			b: &kubeconfig.User{
				Token: "abc123",
			},
			expected: true,
		},
		{
			name: "different tokens",
			a: &kubeconfig.User{
				Token: "abc123",
			},
			b: &kubeconfig.User{
				Token: "def456",
			},
			expected: false,
		},
		{
			name: "identical cert users",
			a: &kubeconfig.User{
				ClientCertificateData: "cert-data",
				ClientKeyData:         "key-data",
			},
			b: &kubeconfig.User{
				ClientCertificateData: "cert-data",
				ClientKeyData:         "key-data",
			},
			expected: true,
		},
		{
			name: "different cert data",
			a: &kubeconfig.User{
				ClientCertificateData: "cert-data-1",
			},
			b: &kubeconfig.User{
				ClientCertificateData: "cert-data-2",
			},
			expected: false,
		},
		{
			name: "identical basic auth users",
			a: &kubeconfig.User{
				Username: "admin",
				Password: "secret",
			},
			b: &kubeconfig.User{
				Username: "admin",
				Password: "secret",
			},
			expected: true,
		},
		{
			name: "different passwords",
			a: &kubeconfig.User{
				Username: "admin",
				Password: "secret1",
			},
			b: &kubeconfig.User{
				Username: "admin",
				Password: "secret2",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usersEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractNameFromConflict(t *testing.T) {
	tests := []struct {
		name     string
		conflict string
		itemType string
		expected string
	}{
		{
			name:     "extract context name",
			conflict: "context 'production-cluster' (different configuration)",
			itemType: "context",
			expected: "production-cluster",
		},
		{
			name:     "extract cluster name",
			conflict: "cluster 'my-cluster' (different server/auth)",
			itemType: "cluster",
			expected: "my-cluster",
		},
		{
			name:     "extract user name",
			conflict: "user 'admin-user' (different credentials)",
			itemType: "user",
			expected: "admin-user",
		},
		{
			name:     "no match found",
			conflict: "some other text",
			itemType: "context",
			expected: "",
		},
		{
			name:     "malformed conflict string",
			conflict: "context without closing quote",
			itemType: "context",
			expected: "",
		},
		{
			name:     "context name with special chars",
			conflict: "context 'my-special-cluster_2023' (different configuration)",
			itemType: "context",
			expected: "my-special-cluster_2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNameFromConflict(tt.conflict, tt.itemType)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCreateSelectiveBackup(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create test kubeconfig
	testConfig := &kubeconfig.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Contexts: []kubeconfig.NamedContext{
			{Name: "context1", Context: &kubeconfig.Context{Cluster: "cluster1", User: "user1"}},
			{Name: "context2", Context: &kubeconfig.Context{Cluster: "cluster2", User: "user2"}},
		},
		Clusters: []kubeconfig.NamedCluster{
			{Name: "cluster1", Cluster: &kubeconfig.Cluster{Server: "https://cluster1.com"}},
			{Name: "cluster2", Cluster: &kubeconfig.Cluster{Server: "https://cluster2.com"}},
		},
		Users: []kubeconfig.NamedUser{
			{Name: "user1", User: &kubeconfig.User{Token: "token1"}},
			{Name: "user2", User: &kubeconfig.User{Token: "token2"}},
		},
	}

	err := kubeconfig.Save(testConfig, kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to save test kubeconfig: %v", err)
	}

	tests := []struct {
		name              string
		shouldContainCtx  string
		shouldContainUser string
		conflicts         []string
		expectedContexts  int
		expectedClusters  int
		expectedUsers     int
	}{
		{
			name:              "single context conflict",
			conflicts:         []string{"context 'context1' (different configuration)"},
			expectedContexts:  1,
			expectedClusters:  1, // cluster1 is included because context1 references it
			expectedUsers:     1, // user1 is included because context1 references it
			shouldContainCtx:  "context1",
			shouldContainUser: "user1",
		},
		{
			name:              "user conflict only",
			conflicts:         []string{"user 'user2' (different credentials)"},
			expectedContexts:  0,
			expectedClusters:  0,
			expectedUsers:     1,
			shouldContainUser: "user2",
		},
		{
			name:              "multiple conflicts",
			conflicts:         []string{"context 'context1' (different configuration)", "user 'user2' (different credentials)"},
			expectedContexts:  1,
			expectedClusters:  1,
			expectedUsers:     2, // user1 (from context1) + user2 (direct conflict)
			shouldContainCtx:  "context1",
			shouldContainUser: "user2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(false, true) // quiet logger
			backupPath, err := createSelectiveBackup(kubeconfigPath, tt.conflicts, log)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify backup file creation
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				t.Errorf("Backup file was not created: %s", backupPath)
				return
			}

			// Load and verify backup content
			backupConfig, err := kubeconfig.Load(backupPath)
			if err != nil {
				t.Errorf("Failed to load backup: %v", err)
				return
			}

			if len(backupConfig.Contexts) != tt.expectedContexts {
				t.Errorf("Expected %d contexts, got %d", tt.expectedContexts, len(backupConfig.Contexts))
			}

			if len(backupConfig.Clusters) != tt.expectedClusters {
				t.Errorf("Expected %d clusters, got %d", tt.expectedClusters, len(backupConfig.Clusters))
			}

			if len(backupConfig.Users) != tt.expectedUsers {
				t.Errorf("Expected %d users, got %d", tt.expectedUsers, len(backupConfig.Users))
			}

			// Check specific content if specified
			if tt.shouldContainCtx != "" {
				found := false
				for _, ctx := range backupConfig.Contexts {
					if ctx.Name == tt.shouldContainCtx {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected backup to contain context '%s'", tt.shouldContainCtx)
				}
			}

			if tt.shouldContainUser != "" {
				found := false
				for _, user := range backupConfig.Users {
					if user.Name == tt.shouldContainUser {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected backup to contain user '%s'", tt.shouldContainUser)
				}
			}

			// Verify backup filename pattern
			if !strings.Contains(backupPath, ".selective-backup.") {
				t.Errorf("Backup path doesn't contain expected pattern: %s", backupPath)
			}

			// Cleanup
			os.Remove(backupPath)
		})
	}
}

func TestShouldCreateBackupBeforeRestore(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "current-config")
	backupPath := filepath.Join(tmpDir, "backup-config")

	// Create current config
	currentConfig := &kubeconfig.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Contexts: []kubeconfig.NamedContext{
			{Name: "prod", Context: &kubeconfig.Context{Cluster: "prod-cluster", User: "admin"}},
		},
		Clusters: []kubeconfig.NamedCluster{
			{Name: "prod-cluster", Cluster: &kubeconfig.Cluster{Server: "https://prod.com"}},
		},
		Users: []kubeconfig.NamedUser{
			{Name: "admin", User: &kubeconfig.User{Token: "current-token"}},
		},
	}

	err := kubeconfig.Save(currentConfig, currentPath)
	if err != nil {
		t.Fatalf("Failed to save current config: %v", err)
	}

	tests := []struct {
		backupConfig          *kubeconfig.Config
		name                  string
		expectedReason        string
		expectedConflictCount int
		expectedShouldBackup  bool
	}{
		{
			name: "no conflicts - safe merge",
			backupConfig: &kubeconfig.Config{
				APIVersion: "v1",
				Kind:       "Config",
				Contexts: []kubeconfig.NamedContext{
					{Name: "dev", Context: &kubeconfig.Context{Cluster: "dev-cluster", User: "dev-user"}},
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "dev-cluster", Cluster: &kubeconfig.Cluster{Server: "https://dev.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "dev-user", User: &kubeconfig.User{Token: "dev-token"}},
				},
			},
			expectedShouldBackup:  false,
			expectedReason:        "no conflicts detected - backup contexts can be safely merged",
			expectedConflictCount: 0,
		},
		{
			name: "user conflicts detected",
			backupConfig: &kubeconfig.Config{
				APIVersion: "v1",
				Kind:       "Config",
				Contexts: []kubeconfig.NamedContext{
					{Name: "dev", Context: &kubeconfig.Context{Cluster: "dev-cluster", User: "admin"}}, // same user name, will conflict
				},
				Clusters: []kubeconfig.NamedCluster{
					{Name: "dev-cluster", Cluster: &kubeconfig.Cluster{Server: "https://dev.com"}},
				},
				Users: []kubeconfig.NamedUser{
					{Name: "admin", User: &kubeconfig.User{Token: "different-token"}}, // conflict!
				},
			},
			expectedShouldBackup:  false, // Determined by mocked user choice "none"
			expectedReason:        "user chose to proceed without backup",
			expectedConflictCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save backup config
			err := kubeconfig.Save(tt.backupConfig, backupPath)
			if err != nil {
				t.Fatalf("Failed to save backup config: %v", err)
			}

			selectedBackup := Backup{
				Name: "test-backup",
				Path: backupPath,
			}

			log := logger.New(false, true) // quiet logger

			// Mock the user interaction for conflicts
			// We'll need to test this function in parts due to the interactive nature

			// Test just the conflict analysis part
			currentCfg, err := kubeconfig.Load(currentPath)
			if err != nil {
				t.Fatalf("Failed to load current config: %v", err)
			}

			backupCfg, err := kubeconfig.Load(backupPath)
			if err != nil {
				t.Fatalf("Failed to load backup config: %v", err)
			}

			conflicts := analyzeRestoreConflicts(currentCfg, backupCfg, log)

			if len(conflicts) != tt.expectedConflictCount {
				t.Errorf("Expected %d conflicts, got %d: %v", tt.expectedConflictCount, len(conflicts), conflicts)
			}

			// For the no-conflict case, we can test the full function
			if tt.expectedConflictCount == 0 {
				shouldBackup, reason, conflictList := shouldCreateBackupBeforeRestore(currentPath, []Backup{}, selectedBackup, log)

				if shouldBackup != tt.expectedShouldBackup {
					t.Errorf("Expected shouldBackup=%v, got %v", tt.expectedShouldBackup, shouldBackup)
				}

				if reason != tt.expectedReason {
					t.Errorf("Expected reason '%s', got '%s'", tt.expectedReason, reason)
				}

				if len(conflictList) != tt.expectedConflictCount {
					t.Errorf("Expected %d conflicts in return value, got %d", tt.expectedConflictCount, len(conflictList))
				}
			}
		})
	}
}

func TestShouldCreateBackupBeforeRestoreErrorCases(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.New(false, true)

	tests := []struct {
		name           string
		kubeconfigPath string
		backupPath     string
		expectedReason string
		expectedError  bool
	}{
		{
			name:           "current kubeconfig doesn't exist",
			kubeconfigPath: filepath.Join(tmpDir, "nonexistent"),
			backupPath:     "",
			expectedError:  true,
			expectedReason: "could not load current kubeconfig for analysis",
		},
		{
			name:           "backup kubeconfig doesn't exist",
			kubeconfigPath: filepath.Join(tmpDir, "valid-current"),
			backupPath:     filepath.Join(tmpDir, "nonexistent-backup"),
			expectedError:  true,
			expectedReason: "could not load backup kubeconfig for analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a valid current config if needed
			if tt.kubeconfigPath != "" && !strings.Contains(tt.kubeconfigPath, "nonexistent") {
				config := &kubeconfig.Config{APIVersion: "v1", Kind: "Config"}
				kubeconfig.Save(config, tt.kubeconfigPath)
			}

			selectedBackup := Backup{
				Path: tt.backupPath,
			}

			shouldBackup, reason, conflicts := shouldCreateBackupBeforeRestore(tt.kubeconfigPath, []Backup{}, selectedBackup, log)

			if tt.expectedError {
				if shouldBackup != true {
					t.Errorf("Expected shouldBackup=true for error case, got %v", shouldBackup)
				}
				if !strings.Contains(reason, tt.expectedReason) {
					t.Errorf("Expected reason to contain '%s', got '%s'", tt.expectedReason, reason)
				}
				if conflicts != nil {
					t.Errorf("Expected nil conflicts for error case, got %v", conflicts)
				}
			}
		})
	}
}
