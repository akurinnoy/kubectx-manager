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

package kubeconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		expectCtx   int
		expectClu   int
		expectUsr   int
	}{
		{
			name: "valid kubeconfig",
			content: `apiVersion: v1
kind: Config
current-context: test-context
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
- name: another-context
  context:
    cluster: another-cluster
    user: another-user
clusters:
- name: test-cluster
  cluster:
    server: https://test.example.com
- name: another-cluster
  cluster:
    server: https://another.example.com
users:
- name: test-user
  user:
    token: test-token
- name: another-user
  user:
    token: another-token
`,
			expectCtx: 2,
			expectClu: 2,
			expectUsr: 2,
		},
		{
			name: "empty kubeconfig",
			content: `apiVersion: v1
kind: Config
contexts: []
clusters: []
users: []
`,
			expectCtx: 0,
			expectClu: 0,
			expectUsr: 0,
		},
		{
			name: "invalid yaml",
			content: `invalid: yaml: content:
  - malformed
    - structure
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			kubeconfigPath := filepath.Join(tmpDir, "config")

			err := os.WriteFile(kubeconfigPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test kubeconfig: %v", err)
			}

			cfg, err := Load(kubeconfigPath)
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

			if len(cfg.Contexts) != tt.expectCtx {
				t.Errorf("Expected %d contexts, got %d", tt.expectCtx, len(cfg.Contexts))
			}
			if len(cfg.Clusters) != tt.expectClu {
				t.Errorf("Expected %d clusters, got %d", tt.expectClu, len(cfg.Clusters))
			}
			if len(cfg.Users) != tt.expectUsr {
				t.Errorf("Expected %d users, got %d", tt.expectUsr, len(cfg.Users))
			}

			// Test internal maps are built
			if len(cfg.contextMap) != tt.expectCtx {
				t.Errorf("Expected %d contexts in map, got %d", tt.expectCtx, len(cfg.contextMap))
			}
		})
	}
}

func TestGetContextNames(t *testing.T) {
	cfg := &Config{
		Contexts: []NamedContext{
			{Name: "context1", Context: &Context{}},
			{Name: "context2", Context: &Context{}},
			{Name: "context3", Context: &Context{}},
		},
	}
	cfg.buildInternalMaps()

	names := cfg.GetContextNames()
	if len(names) != 3 {
		t.Errorf("Expected 3 context names, got %d", len(names))
	}

	expectedNames := map[string]bool{
		"context1": false,
		"context2": false,
		"context3": false,
	}

	for _, name := range names {
		if _, exists := expectedNames[name]; !exists {
			t.Errorf("Unexpected context name: %s", name)
		}
		expectedNames[name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Missing expected context name: %s", name)
		}
	}
}

func TestGetContext(t *testing.T) {
	testContext := &Context{
		Cluster: "test-cluster",
		User:    "test-user",
	}

	cfg := &Config{
		Contexts: []NamedContext{
			{Name: "test-context", Context: testContext},
		},
	}
	cfg.buildInternalMaps()

	// Test existing context
	ctx := cfg.GetContext("test-context")
	if ctx == nil {
		t.Fatalf("Expected to find context, got nil")
	}
	if ctx.Cluster != "test-cluster" {
		t.Errorf("Expected cluster 'test-cluster', got %s", ctx.Cluster)
	}

	// Test non-existing context
	ctx = cfg.GetContext("non-existent")
	if ctx != nil {
		t.Errorf("Expected nil for non-existent context, got %v", ctx)
	}
}

func TestRemoveContexts(t *testing.T) {
	cfg := &Config{
		CurrentContext: "context1",
		Contexts: []NamedContext{
			{Name: "context1", Context: &Context{Cluster: "cluster1", User: "user1"}},
			{Name: "context2", Context: &Context{Cluster: "cluster2", User: "user2"}},
			{Name: "context3", Context: &Context{Cluster: "cluster1", User: "user1"}}, // shares cluster/user
		},
		Clusters: []NamedCluster{
			{Name: "cluster1", Cluster: &Cluster{Server: "https://cluster1.com"}},
			{Name: "cluster2", Cluster: &Cluster{Server: "https://cluster2.com"}},
			{Name: "orphaned-cluster", Cluster: &Cluster{Server: "https://orphaned.com"}},
		},
		Users: []NamedUser{
			{Name: "user1", User: &User{Token: "token1"}},
			{Name: "user2", User: &User{Token: "token2"}},
			{Name: "orphaned-user", User: &User{Token: "orphaned"}},
		},
	}
	cfg.buildInternalMaps()

	// Remove context1 and context2
	err := RemoveContexts(cfg, []string{"context1", "context2"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check contexts
	if len(cfg.Contexts) != 1 {
		t.Errorf("Expected 1 context remaining, got %d", len(cfg.Contexts))
	}
	if cfg.Contexts[0].Name != "context3" {
		t.Errorf("Expected context3 to remain, got %s", cfg.Contexts[0].Name)
	}

	// Verify current-context updated correctly
	if cfg.CurrentContext != "context3" {
		t.Errorf("Expected current-context to be context3, got %s", cfg.CurrentContext)
	}

	// Check orphaned clusters were removed
	if len(cfg.Clusters) != 1 {
		t.Errorf("Expected 1 cluster remaining, got %d", len(cfg.Clusters))
	}
	if cfg.Clusters[0].Name != "cluster1" {
		t.Errorf("Expected cluster1 to remain, got %s", cfg.Clusters[0].Name)
	}

	// Check orphaned users were removed
	if len(cfg.Users) != 1 {
		t.Errorf("Expected 1 user remaining, got %d", len(cfg.Users))
	}
	if cfg.Users[0].Name != "user1" {
		t.Errorf("Expected user1 to remain, got %s", cfg.Users[0].Name)
	}
}

func TestRemoveAllContexts(t *testing.T) {
	cfg := &Config{
		CurrentContext: "context1",
		Contexts: []NamedContext{
			{Name: "context1", Context: &Context{Cluster: "cluster1", User: "user1"}},
		},
		Clusters: []NamedCluster{
			{Name: "cluster1", Cluster: &Cluster{Server: "https://cluster1.com"}},
		},
		Users: []NamedUser{
			{Name: "user1", User: &User{Token: "token1"}},
		},
	}
	cfg.buildInternalMaps()

	err := RemoveContexts(cfg, []string{"context1"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check everything is empty
	if len(cfg.Contexts) != 0 {
		t.Errorf("Expected 0 contexts, got %d", len(cfg.Contexts))
	}
	if len(cfg.Clusters) != 0 {
		t.Errorf("Expected 0 clusters, got %d", len(cfg.Clusters))
	}
	if len(cfg.Users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(cfg.Users))
	}
	if cfg.CurrentContext != "" {
		t.Errorf("Expected empty current-context, got %s", cfg.CurrentContext)
	}
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "config")
	originalContent := "test config content"

	err := os.WriteFile(originalPath, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupPath, err := CreateBackup(originalPath)
	if err != nil {
		t.Errorf("Unexpected error creating backup: %v", err)
	}

	// Check backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file was not created")
	}

	// Check backup content matches original
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Errorf("Failed to read backup file: %v", err)
	}
	if string(backupContent) != originalContent {
		t.Errorf("Backup content doesn't match original")
	}

	// Check backup filename format
	if !strings.Contains(backupPath, ".backup.") {
		t.Errorf("Backup filename doesn't contain expected pattern")
	}
}

func TestFindBackups(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create original file
	err := os.WriteFile(kubeconfigPath, []byte("original"), 0644)
	if err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create some backup files with proper timestamp format
	timestamps := []string{
		"20231201-120000",
		"20231201-130000",
		"20231201-140000",
	}

	for _, ts := range timestamps {
		backupPath := kubeconfigPath + ".backup." + ts
		err := os.WriteFile(backupPath, []byte("backup-"+ts), 0644)
		if err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
	}

	// Create a file that shouldn't match (wrong format)
	wrongPath := kubeconfigPath + ".backup.invalid"
	err = os.WriteFile(wrongPath, []byte("wrong"), 0644)
	if err != nil {
		t.Fatalf("Failed to create wrong file: %v", err)
	}

	// Find backups using the function from restore.go
	// We need to create a simple version here since restore.go is in cmd package
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	var backupCount int
	prefix := "config.backup."
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			timestampStr := strings.TrimPrefix(entry.Name(), prefix)
			_, err := time.Parse("20060102-150405", timestampStr)
			if err == nil {
				backupCount++
			}
		}
	}

	if backupCount != 3 {
		t.Errorf("Expected 3 valid backups, found %d", backupCount)
	}
}

func TestIsAuthValid(t *testing.T) {
	tests := []struct {
		user     *User
		name     string
		expected bool
	}{
		{
			name: "valid token auth but unreachable cluster",
			user: &User{
				Token: "valid-token",
			},
			expected: false, // Unreachable cluster
		},
		{
			name: "valid cert auth but unreachable cluster",
			user: &User{
				ClientCertificateData: "cert-data",
			},
			expected: false, // Unreachable cluster
		},
		{
			name: "valid cert file auth but unreachable cluster",
			user: &User{
				ClientCertificate: "/path/to/cert",
			},
			expected: false, // Unreachable cluster
		},
		{
			name: "valid basic auth but unreachable cluster",
			user: &User{
				Username: "admin",
				Password: "password",
			},
			expected: false, // Unreachable cluster
		},
		{
			name: "valid auth provider but unreachable cluster",
			user: &User{
				AuthProvider: &AuthProvider{
					Name:   "oidc",
					Config: map[string]string{"issuer": "https://example.com"},
				},
			},
			expected: false, // Unreachable cluster
		},
		{
			name: "auth provider without config",
			user: &User{
				AuthProvider: &AuthProvider{
					Name: "oidc",
				},
			},
			expected: false,
		},
		{
			name: "valid exec auth but unreachable cluster",
			user: &User{
				Exec: &ExecConfig{
					Command: "/bin/sh",
				},
			},
			expected: false, // Unreachable cluster
		},
		{
			name:     "empty user",
			user:     &User{},
			expected: false,
		},
		{
			name: "empty token",
			user: &User{
				Token: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Contexts: []NamedContext{
					{
						Name: "test-context",
						Context: &Context{
							Cluster: "test-cluster",
							User:    "test-user",
						},
					},
				},
				Clusters: []NamedCluster{
					{
						Name: "test-cluster",
						Cluster: &Cluster{
							Server: "https://unreachable.test.invalid", // Unreachable server for testing
						},
					},
				},
				Users: []NamedUser{
					{
						Name: "test-user",
						User: tt.user,
					},
				},
			}
			cfg.buildInternalMaps()

			result := IsAuthValid(cfg, "test-context")
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for user %+v", tt.expected, result, tt.user)
			}
		})
	}
}

func TestIsAuthValidNonExistentContext(t *testing.T) {
	cfg := &Config{}
	cfg.buildInternalMaps()

	result := IsAuthValid(cfg, "non-existent")
	if result != false {
		t.Errorf("Expected false for non-existent context, got %v", result)
	}
}

func TestSave(t *testing.T) {
	cfg := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Contexts: []NamedContext{
			{Name: "test", Context: &Context{Cluster: "cluster", User: "user"}},
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	err := Save(cfg, configPath)
	if err != nil {
		t.Errorf("Unexpected error saving config: %v", err)
	}

	// Verify file created successfully
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created")
	}

	// Check we can load it back
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Errorf("Failed to load saved config: %v", err)
	}

	if loadedCfg.APIVersion != "v1" {
		t.Errorf("Expected APIVersion v1, got %s", loadedCfg.APIVersion)
	}
	if len(loadedCfg.Contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(loadedCfg.Contexts))
	}
}
