// Package config provides configuration management for kubectx-manager.
// It handles reading and writing configuration files and ignore patterns.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Whitelist []string `yaml:"whitelist"`
	patterns  []*regexp.Regexp
}

// Load reads the configuration file and compiles patterns
func Load(configPath string) (*Config, error) {
	cfg := &Config{}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// Read config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error if needed, but don't fail the operation
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		cfg.Whitelist = append(cfg.Whitelist, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Compile patterns
	for _, pattern := range cfg.Whitelist {
		regex, err := compilePattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
		}
		cfg.patterns = append(cfg.patterns, regex)
	}

	return cfg, nil
}

// MatchesWhitelist checks if a context name matches any whitelist pattern
func (c *Config) MatchesWhitelist(contextName string) bool {
	for _, pattern := range c.patterns {
		if pattern.MatchString(contextName) {
			return true
		}
	}
	return false
}

// compilePattern converts a glob-like pattern to a regex
func compilePattern(pattern string) (*regexp.Regexp, error) {
	// Escape special regex characters except * and ?
	escaped := regexp.QuoteMeta(pattern)

	// Convert glob patterns to regex
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	escaped = strings.ReplaceAll(escaped, `\?`, ".")

	// Anchor the pattern to match the entire string
	escaped = "^" + escaped + "$"

	return regexp.Compile(escaped)
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	defaultContent := `# kubectx-manager ignore file (contexts to keep)
# List context patterns to keep (whitelist)
# Supports glob patterns: * (any characters) and ? (single character)
# Examples:
# production-*
# staging-cluster
# *-important
# my-dev-context

# Add your patterns below (one per line):
`

	return os.WriteFile(configPath, []byte(defaultContent), 0644)
}
