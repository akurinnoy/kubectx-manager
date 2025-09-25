package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    []string
		expectError bool
	}{
		{
			name: "valid config with patterns",
			content: `# Test config
production-*
staging-cluster
*-important
my-dev-context
`,
			expected: []string{"production-*", "staging-cluster", "*-important", "my-dev-context"},
		},
		{
			name: "config with comments and empty lines",
			content: `# This is a comment
production-*

# Another comment
staging-cluster
# Empty line above
*-important

my-dev-context
`,
			expected: []string{"production-*", "staging-cluster", "*-important", "my-dev-context"},
		},
		{
			name:     "empty config",
			content:  "",
			expected: []string{},
		},
		{
			name: "config with only comments",
			content: `# Only comments
# No patterns here
`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")

			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			cfg, err := Load(configPath)
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

			if len(cfg.Whitelist) != len(tt.expected) {
				t.Errorf("Expected %d patterns, got %d", len(tt.expected), len(cfg.Whitelist))
				return
			}

			for i, expected := range tt.expected {
				if cfg.Whitelist[i] != expected {
					t.Errorf("Pattern %d: expected %q, got %q", i, expected, cfg.Whitelist[i])
				}
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")

	cfg, err := Load(configPath)
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
		return
	}

	// Should create default config
	if len(cfg.Whitelist) != 0 {
		t.Errorf("Expected empty whitelist for default config, got %d patterns", len(cfg.Whitelist))
	}

	// Verify default config file creation
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Expected default config file to be created")
	}
}

func TestMatchesWhitelist(t *testing.T) {
	tests := []struct {
		name        string
		contextName string
		patterns    []string
		expected    bool
	}{
		{
			name:        "exact match",
			patterns:    []string{"production-cluster"},
			contextName: "production-cluster",
			expected:    true,
		},
		{
			name:        "wildcard match - prefix",
			patterns:    []string{"production-*"},
			contextName: "production-cluster",
			expected:    true,
		},
		{
			name:        "wildcard match - suffix",
			patterns:    []string{"*-production"},
			contextName: "my-production",
			expected:    true,
		},
		{
			name:        "wildcard match - middle",
			patterns:    []string{"prod-*-cluster"},
			contextName: "prod-east-cluster",
			expected:    true,
		},
		{
			name:        "question mark match",
			patterns:    []string{"cluster-?"},
			contextName: "cluster-1",
			expected:    true,
		},
		{
			name:        "question mark no match - too many chars",
			patterns:    []string{"cluster-?"},
			contextName: "cluster-10",
			expected:    false,
		},
		{
			name:        "no match",
			patterns:    []string{"production-*"},
			contextName: "staging-cluster",
			expected:    false,
		},
		{
			name:        "multiple patterns - first matches",
			patterns:    []string{"production-*", "staging-*"},
			contextName: "production-cluster",
			expected:    true,
		},
		{
			name:        "multiple patterns - second matches",
			patterns:    []string{"production-*", "staging-*"},
			contextName: "staging-cluster",
			expected:    true,
		},
		{
			name:        "multiple patterns - none match",
			patterns:    []string{"production-*", "staging-*"},
			contextName: "development-cluster",
			expected:    false,
		},
		{
			name:        "empty patterns",
			patterns:    []string{},
			contextName: "any-context",
			expected:    false,
		},
		{
			name:        "case sensitive",
			patterns:    []string{"Production-*"},
			contextName: "production-cluster",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Whitelist: tt.patterns}

			// Compile patterns
			for _, pattern := range tt.patterns {
				regex, err := compilePattern(pattern)
				if err != nil {
					t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
				}
				cfg.patterns = append(cfg.patterns, regex)
			}

			result := cfg.MatchesWhitelist(tt.contextName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for context %q with patterns %v",
					tt.expected, result, tt.contextName, tt.patterns)
			}
		})
	}
}

func TestCompilePattern(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		testString  string
		shouldMatch bool
		expectError bool
	}{
		{
			name:        "simple wildcard",
			pattern:     "test-*",
			testString:  "test-cluster",
			shouldMatch: true,
		},
		{
			name:        "wildcard no match",
			pattern:     "test-*",
			testString:  "prod-cluster",
			shouldMatch: false,
		},
		{
			name:        "question mark",
			pattern:     "test-?",
			testString:  "test-1",
			shouldMatch: true,
		},
		{
			name:        "question mark no match",
			pattern:     "test-?",
			testString:  "test-10",
			shouldMatch: false,
		},
		{
			name:        "exact match",
			pattern:     "exact",
			testString:  "exact",
			shouldMatch: true,
		},
		{
			name:        "partial match fails (anchored)",
			pattern:     "test",
			testString:  "testing",
			shouldMatch: false,
		},
		{
			name:        "special regex chars escaped",
			pattern:     "test.cluster",
			testString:  "test.cluster",
			shouldMatch: true,
		},
		{
			name:        "special regex chars escaped - dot doesn't match any",
			pattern:     "test.cluster",
			testString:  "testXcluster",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := compilePattern(tt.pattern)
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

			matches := regex.MatchString(tt.testString)
			if matches != tt.shouldMatch {
				t.Errorf("Pattern %q with string %q: expected match=%v, got %v",
					tt.pattern, tt.testString, tt.shouldMatch, matches)
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")

	err := createDefaultConfig(configPath)
	if err != nil {
		t.Errorf("Failed to create default config: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Default config file was not created")
	}

	// Check file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Errorf("Failed to read default config file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "kubectx-manager ignore file") {
		t.Errorf("Default config doesn't contain expected header")
	}
	if !strings.Contains(contentStr, "production-*") {
		t.Errorf("Default config doesn't contain example patterns")
	}
}

func TestLoadInvalidPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kubectx-manager_ignore")

	// Create file and make it unreadable
	err := os.WriteFile(configPath, []byte("test"), 0000)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Errorf("Expected error for unreadable file, but got none")
	}
}
