package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	// Test that the root command can be created without errors
	cmd := &cobra.Command{
		Use:   "kubectx-manager",
		Short: "Clean up Kubernetes contexts from your kubeconfig",
		RunE:  runCleanup,
	}

	if cmd.Use != "kubectx-manager" {
		t.Errorf("Expected command name 'kubectx-manager', got %s", cmd.Use)
	}
	if cmd.Short != "Clean up Kubernetes contexts from your kubeconfig" {
		t.Errorf("Expected command short description, got %s", cmd.Short)
	}
	if cmd.RunE == nil {
		t.Error("Expected RunE function to be set")
	}
}

func TestFindContextsToRemove(t *testing.T) {
	// Create a mock config for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")
	err := os.WriteFile(configPath, []byte("production-*\nstaging-cluster\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create a mock kubeconfig
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: production-cluster
contexts:
- name: production-cluster
  context:
    cluster: prod-cluster
    user: prod-user
- name: production-backup
  context:
    cluster: prod-backup-cluster
    user: prod-backup-user
- name: staging-cluster
  context:
    cluster: stage-cluster
    user: stage-user
- name: development-cluster
  context:
    cluster: dev-cluster
    user: dev-user
- name: test-cluster
  context:
    cluster: test-cluster
    user: test-user
clusters:
- name: prod-cluster
  cluster:
    server: https://prod.example.com
- name: prod-backup-cluster
  cluster:
    server: https://prod-backup.example.com
- name: stage-cluster
  cluster:
    server: https://stage.example.com
- name: dev-cluster
  cluster:
    server: https://dev.example.com
- name: test-cluster
  cluster:
    server: https://test.example.com
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
`

	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig: %v", err)
	}

	// Test the function (we'll need to import the necessary packages)
	// Test command execution
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Capture output
	var output bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up command args for dry-run
	os.Args = []string{"kubectx-manager", "--dry-run", "--config", configPath, "--kubeconfig", kubeconfigPath}

	// Reset flags for testing
	dryRun = false
	authCheck = false
	verbose = false
	quiet = false
	interactive = false
	configFile = ""
	kubeConfig = ""

	// Execute root command
	err = Execute()

	w.Close()
	os.Stdout = oldStdout
	output.ReadFrom(r)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	outputStr := output.String()

	// Check that it identifies the correct contexts to remove
	// Should remove: development-cluster, test-cluster
	// Should keep: production-cluster, production-backup, staging-cluster
	if !strings.Contains(outputStr, "development-cluster") {
		t.Errorf("Expected to remove development-cluster, but it's not in output: %s", outputStr)
	}
	if !strings.Contains(outputStr, "test-cluster") {
		t.Errorf("Expected to remove test-cluster, but it's not in output: %s", outputStr)
	}
	if strings.Contains(outputStr, "production-cluster") {
		t.Errorf("Should not remove production-cluster (matches production-*), but it's in output: %s", outputStr)
	}
	if strings.Contains(outputStr, "staging-cluster") {
		t.Errorf("Should not remove staging-cluster (exact match), but it's in output: %s", outputStr)
	}
}

func TestConfirmRemoval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes lowercase", "y\n", true},
		{"yes uppercase", "Y\n", true},
		{"yes full", "yes\n", true},
		{"yes full capitalized", "Yes\n", true},
		{"no lowercase", "n\n", false},
		{"no uppercase", "N\n", false},
		{"no full", "no\n", false},
		{"empty", "\n", false},
		{"random text", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Write input
			go func() {
				defer w.Close()
				w.WriteString(tt.input)
			}()

			result := confirmRemoval([]string{"test-context"})

			os.Stdin = oldStdin

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for input %q", tt.expected, result, tt.input)
			}
		})
	}
}

func TestFlagsInitialization(t *testing.T) {
	// Create a new command to test flag initialization
	testCmd := &cobra.Command{
		Use: "test",
	}

	homeDir, _ := os.UserHomeDir()
	defaultConfig := filepath.Join(homeDir, ".kubectx-manager_ignore")
	defaultKubeConfig := filepath.Join(homeDir, ".kube", "config")

	testCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show what would be removed without making changes")
	testCmd.Flags().BoolVarP(&authCheck, "auth-check", "a", false, "Remove contexts with expired or unreachable authentication")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose (debug) output")
	testCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
	testCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Prompt for confirmation before removing contexts")
	testCmd.Flags().StringVarP(&configFile, "config", "c", defaultConfig, "Path to kubectx-manager configuration file")
	testCmd.Flags().StringVarP(&kubeConfig, "kubeconfig", "k", defaultKubeConfig, "Path to kubeconfig file")

	// Test flag defaults
	flag := testCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dry-run flag not found")
	}
	if flag.DefValue != "false" {
		t.Errorf("Expected dry-run default to be 'false', got %s", flag.DefValue)
	}

	flag = testCmd.Flags().Lookup("interactive")
	if flag == nil {
		t.Fatal("interactive flag not found")
	}
	if flag.DefValue != "false" {
		t.Errorf("Expected interactive default to be 'false', got %s", flag.DefValue)
	}

	flag = testCmd.Flags().Lookup("config")
	if flag == nil {
		t.Fatal("config flag not found")
	}
	if !strings.Contains(flag.DefValue, ".kubectx-manager_ignore") {
		t.Errorf("Expected config default to contain '.kubectx-manager_ignore', got %s", flag.DefValue)
	}
}

func TestNoInteractiveDefault(t *testing.T) {
	// Test that interactive is false by default (no prompts by default)
	if interactive != false {
		t.Errorf("Expected interactive to default to false, got %v", interactive)
	}
}

func TestEmptyContextList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty config
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")
	err := os.WriteFile(configPath, []byte("# No patterns\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create kubeconfig with no contexts
	kubeconfigContent := `apiVersion: v1
kind: Config
contexts: []
clusters: []
users: []
`
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig: %v", err)
	}

	// Test with empty kubeconfig
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	var output bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	os.Args = []string{"kubectx-manager", "--dry-run", "--config", configPath, "--kubeconfig", kubeconfigPath}

	// Reset flags
	dryRun = false
	configFile = ""
	kubeConfig = ""

	err = Execute()

	w.Close()
	os.Stdout = oldStdout
	output.ReadFrom(r)

	if err != nil {
		t.Errorf("Unexpected error with empty kubeconfig: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "No contexts to remove") {
		t.Errorf("Expected 'No contexts to remove' message, got: %s", outputStr)
	}
}
