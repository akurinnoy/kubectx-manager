package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create test config file
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")
	configContent := `# Test configuration
production-*
staging-cluster
*-important
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create test kubeconfig
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: production-cluster
contexts:
- name: production-cluster
  context:
    cluster: prod
    user: prod-user
- name: production-backup
  context:
    cluster: prod-backup
    user: prod-backup-user
- name: staging-cluster
  context:
    cluster: stage
    user: stage-user
- name: development-cluster
  context:
    cluster: dev
    user: dev-user
- name: test-cluster
  context:
    cluster: test
    user: test-user
- name: my-important
  context:
    cluster: important
    user: important-user
clusters:
- name: prod
  cluster:
    server: https://prod.example.com
- name: prod-backup
  cluster:
    server: https://prod-backup.example.com
- name: stage
  cluster:
    server: https://stage.example.com
- name: dev
  cluster:
    server: https://dev.example.com
- name: test
  cluster:
    server: https://test.example.com
- name: important
  cluster:
    server: https://important.example.com
users:
- name: prod-user
  user:
    token: prod-token
- name: prod-backup-user
  user:
    token: prod-backup-token
- name: stage-user
  user:
    token: stage-token
- name: dev-user
  user:
    token: dev-token
- name: test-user
  user:
    token: test-token
- name: important-user
  user:
    token: important-token
`
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "kubectx-manager")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Run dry-run test
	cmd = exec.Command(binaryPath, "--dry-run", "--verbose",
		"--config", configPath, "--kubeconfig", kubeconfigPath)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output.String())
	}

	outputStr := output.String()

	// Verify expected contexts to remove
	expectedToRemove := []string{"development-cluster", "test-cluster"}
	for _, ctx := range expectedToRemove {
		if !strings.Contains(outputStr, ctx) {
			t.Errorf("Expected to remove %s, but not found in output: %s", ctx, outputStr)
		}
	}

	// Verify contexts to keep are NOT in removal list
	expectedToKeep := []string{"production-cluster", "production-backup", "staging-cluster", "my-important"}
	for _, ctx := range expectedToKeep {
		// Check that these don't appear in "Contexts to remove:" section
		lines := strings.Split(outputStr, "\n")
		inRemovalSection := false
		for _, line := range lines {
			if strings.Contains(line, "Contexts to remove:") {
				inRemovalSection = true
				continue
			}
			if inRemovalSection && strings.Contains(line, "Dry run mode") {
				inRemovalSection = false
				break
			}
			if inRemovalSection && strings.Contains(line, ctx) {
				t.Errorf("Context %s should be kept but appears in removal list: %s", ctx, outputStr)
			}
		}
	}

	// Verify dry-run message
	if !strings.Contains(outputStr, "Dry run mode - no changes made") {
		t.Errorf("Expected dry-run message not found in output: %s", outputStr)
	}

	// Verify debug output
	if !strings.Contains(outputStr, "[DEBUG]") {
		t.Errorf("Expected debug output with --verbose flag: %s", outputStr)
	}
}

func TestIntegrationActualRemoval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create test config file
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")
	configContent := `production-cluster
staging-*
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create test kubeconfig
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	originalKubeconfig := `apiVersion: v1
kind: Config
current-context: production-cluster
contexts:
- name: production-cluster
  context:
    cluster: prod
    user: prod-user
- name: staging-east
  context:
    cluster: stage-east
    user: stage-user
- name: development-cluster
  context:
    cluster: dev
    user: dev-user
clusters:
- name: prod
  cluster:
    server: https://prod.example.com
- name: stage-east
  cluster:
    server: https://stage-east.example.com
- name: dev
  cluster:
    server: https://dev.example.com
users:
- name: prod-user
  user:
    token: prod-token
- name: stage-user
  user:
    token: stage-token
- name: dev-user
  user:
    token: dev-token
`
	err = os.WriteFile(kubeconfigPath, []byte(originalKubeconfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "kubectx-manager")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Run actual cleanup (non-interactive by default)
	cmd = exec.Command(binaryPath, "--config", configPath, "--kubeconfig", kubeconfigPath)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output.String())
	}

	// Verify backup creation
	backupFiles := []string{}
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), "kubeconfig.backup.") {
			backupFiles = append(backupFiles, entry.Name())
		}
	}

	if len(backupFiles) != 1 {
		t.Errorf("Expected 1 backup file, found %d: %v", len(backupFiles), backupFiles)
	}

	// Read modified kubeconfig
	modifiedKubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to read modified kubeconfig: %v", err)
	}

	modifiedStr := string(modifiedKubeconfig)

	// Verify development-cluster removal
	if strings.Contains(modifiedStr, "development-cluster") {
		t.Errorf("development-cluster should have been removed but still exists")
	}

	// Verify production-cluster and staging-east were kept
	if !strings.Contains(modifiedStr, "production-cluster") {
		t.Errorf("production-cluster should have been kept but was removed")
	}
	if !strings.Contains(modifiedStr, "staging-east") {
		t.Errorf("staging-east should have been kept but was removed")
	}

	// Verify orphaned dev cluster and user were removed
	if strings.Contains(modifiedStr, "name: dev") {
		t.Errorf("Orphaned dev cluster should have been removed")
	}
	if strings.Contains(modifiedStr, "name: dev-user") {
		t.Errorf("Orphaned dev-user should have been removed")
	}

	// Verify current-context is still valid
	if strings.Contains(modifiedStr, `current-context: production-cluster`) {
		// Current context should remain unchanged
	} else {
		t.Errorf("current-context should remain as production-cluster")
	}
}

func TestIntegrationRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	// Create original kubeconfig
	originalContent := `apiVersion: v1
kind: Config
current-context: test-context
contexts:
- name: test-context
  context:
    cluster: test
    user: test-user
clusters:
- name: test
  cluster:
    server: https://test.example.com
users:
- name: test-user
  user:
    token: test-token
`
	err := os.WriteFile(kubeconfigPath, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Create a backup file manually
	backupPath := kubeconfigPath + ".backup.20231201-120000"
	backupContent := `apiVersion: v1
kind: Config
current-context: backup-context
contexts:
- name: backup-context
  context:
    cluster: backup
    user: backup-user
clusters:
- name: backup
  cluster:
    server: https://backup.example.com
users:
- name: backup-user
  user:
    token: backup-token
`
	err = os.WriteFile(backupPath, []byte(backupContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "kubectx-manager")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test restore command (just list backups)
	cmd = exec.Command(binaryPath, "restore", "--kubeconfig", kubeconfigPath)

	// Provide input to cancel the restore
	cmd.Stdin = strings.NewReader("0\n")

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Restore command failed: %v\nOutput: %s", err, output.String())
	}

	outputStr := output.String()

	// Verify backup discovery and listing
	if !strings.Contains(outputStr, "Available backups:") {
		t.Errorf("Expected backup listing, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "kubeconfig.backup.20231201-120000") {
		t.Errorf("Expected backup file to be listed, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "2023-12-01 12:00:00") {
		t.Errorf("Expected formatted timestamp, got: %s", outputStr)
	}
}

func TestIntegrationAuthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create empty config (no whitelist)
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")
	err := os.WriteFile(configPath, []byte("# No whitelist patterns\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create kubeconfig with various auth types
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: valid-token-context
contexts:
- name: valid-token-context
  context:
    cluster: cluster1
    user: valid-token-user
- name: empty-user-context
  context:
    cluster: cluster2
    user: empty-user
- name: cert-user-context
  context:
    cluster: cluster3
    user: cert-user
clusters:
- name: cluster1
  cluster:
    server: https://cluster1.example.com
- name: cluster2
  cluster:
    server: https://cluster2.example.com
- name: cluster3
  cluster:
    server: https://cluster3.example.com
users:
- name: valid-token-user
  user:
    token: valid-token
- name: empty-user
  user: {}
- name: cert-user
  user:
    client-certificate-data: Y2VydGRhdGE=
`
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "kubectx-manager")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Run with auth-check flag
	cmd = exec.Command(binaryPath, "--auth-check", "--dry-run", "--verbose",
		"--config", configPath, "--kubeconfig", kubeconfigPath)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Auth check command failed: %v\nOutput: %s", err, output.String())
	}

	outputStr := output.String()

	// With --auth-check, only contexts with invalid auth should be removed
	// empty-user-context should be marked for removal (no auth)
	// valid-token-user and cert-user should be kept (valid auth)

	// Check debug output for auth validation
	if !strings.Contains(outputStr, "has valid auth") {
		t.Errorf("Expected auth validation debug output: %s", outputStr)
	}
}

func TestIntegrationQuietMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create simple config and kubeconfig
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")
	err := os.WriteFile(configPath, []byte("keep-this\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	kubeconfigContent := `apiVersion: v1
kind: Config
contexts:
- name: remove-this
  context:
    cluster: test
    user: test
clusters:
- name: test
  cluster:
    server: https://test.com
users:
- name: test
  user:
    token: token
`
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig: %v", err)
	}

	// Build binary
	binaryPath := filepath.Join(tmpDir, "kubectx-manager")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Run in quiet mode
	cmd = exec.Command(binaryPath, "--quiet", "--dry-run",
		"--config", configPath, "--kubeconfig", kubeconfigPath)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Quiet mode command failed: %v\nOutput: %s", err, output.String())
	}

	outputStr := output.String()

	// In quiet mode, should have minimal output (no debug, no info about contexts)
	if strings.Contains(outputStr, "Contexts to remove:") {
		t.Errorf("Quiet mode should not show context removal details: %s", outputStr)
	}
	if strings.Contains(outputStr, "[DEBUG]") {
		t.Errorf("Quiet mode should not show debug output: %s", outputStr)
	}
}
