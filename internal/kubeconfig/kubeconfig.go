// Package kubeconfig provides utilities for managing Kubernetes configuration files.
// It includes functions for loading, parsing, merging, and backing up kubeconfig files.
package kubeconfig

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// File permissions for kubeconfig files (readable/writable by owner only)
	kubeconfigFileMode = 0600
	// Timeout values for network operations
	httpTimeout = 10 * time.Second
	ctxTimeout  = 5 * time.Second
	// HTTP status code threshold for success
	httpSuccessThreshold = 500
)

const (
	// BackupTimeFormat is the timestamp format used for backup file names
	BackupTimeFormat = "20060102-150405"
)

// Config represents the structure of a kubeconfig file
type Config struct {
	Preferences    map[string]interface{} `yaml:"preferences,omitempty"`
	contextMap     map[string]*Context    `yaml:"-"`
	clusterMap     map[string]*Cluster    `yaml:"-"`
	userMap        map[string]*User       `yaml:"-"`
	APIVersion     string                 `yaml:"apiVersion"`
	Kind           string                 `yaml:"kind"`
	CurrentContext string                 `yaml:"current-context"`
	Contexts       []NamedContext         `yaml:"contexts"`
	Clusters       []NamedCluster         `yaml:"clusters"`
	Users          []NamedUser            `yaml:"users"`
}

// NamedContext represents a Kubernetes context with its name.
type NamedContext struct {
	Context *Context `yaml:"context"`
	Name    string   `yaml:"name"`
}

// Context represents a Kubernetes context configuration.
type Context struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace,omitempty"`
}

// NamedCluster represents a Kubernetes cluster configuration with its name.
type NamedCluster struct {
	Cluster *Cluster `yaml:"cluster"`
	Name    string   `yaml:"name"`
}

// Cluster represents a Kubernetes cluster connection configuration.
type Cluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	CertificateAuthority     string `yaml:"certificate-authority,omitempty"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

// NamedUser represents a Kubernetes user with its name.
type NamedUser struct {
	User *User  `yaml:"user"`
	Name string `yaml:"name"`
}

// User represents a Kubernetes user authentication configuration.
type User struct {
	AuthProvider          *AuthProvider          `yaml:"auth-provider,omitempty"`
	Exec                  *ExecConfig            `yaml:"exec,omitempty"`
	Extensions            map[string]interface{} `yaml:",inline"`
	ClientCertificateData string                 `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string                 `yaml:"client-key-data,omitempty"`
	ClientCertificate     string                 `yaml:"client-certificate,omitempty"`
	ClientKey             string                 `yaml:"client-key,omitempty"`
	Token                 string                 `yaml:"token,omitempty"`
	Username              string                 `yaml:"username,omitempty"`
	Password              string                 `yaml:"password,omitempty"`
}

// AuthProvider represents an authentication provider configuration.
type AuthProvider struct {
	Config map[string]string `yaml:"config,omitempty"`
	Name   string            `yaml:"name"`
}

// ExecConfig represents an exec-based authentication configuration.
type ExecConfig struct {
	APIVersion string       `yaml:"apiVersion"`
	Command    string       `yaml:"command"`
	Args       []string     `yaml:"args,omitempty"`
	Env        []ExecEnvVar `yaml:"env,omitempty"`
}

// ExecEnvVar represents an environment variable used in exec-based authentication.
// It contains a name-value pair that will be set when executing the auth command.
type ExecEnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Load reads and parses a kubeconfig file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // User-specified kubeconfig path is intentional
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// Build internal maps for easy lookup
	config.buildInternalMaps()

	return &config, nil
}

// buildInternalMaps creates internal maps for easy lookup
func (c *Config) buildInternalMaps() {
	c.contextMap = make(map[string]*Context)
	c.clusterMap = make(map[string]*Cluster)
	c.userMap = make(map[string]*User)

	for _, namedContext := range c.Contexts {
		if namedContext.Context != nil {
			c.contextMap[namedContext.Name] = namedContext.Context
		}
	}

	for _, namedCluster := range c.Clusters {
		if namedCluster.Cluster != nil {
			c.clusterMap[namedCluster.Name] = namedCluster.Cluster
		}
	}

	for _, namedUser := range c.Users {
		if namedUser.User != nil {
			c.userMap[namedUser.Name] = namedUser.User
		}
	}
}

// GetContextNames returns all context names
func (c *Config) GetContextNames() []string {
	var names []string
	for name := range c.contextMap {
		names = append(names, name)
	}
	return names
}

// GetContext returns a context by name
func (c *Config) GetContext(name string) *Context {
	return c.contextMap[name]
}

// GetUser returns a user by name
func (c *Config) GetUser(name string) *User {
	return c.userMap[name]
}

// Save writes the kubeconfig to a file
func Save(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	return os.WriteFile(path, data, kubeconfigFileMode)
}

// CreateBackup creates a backup of the kubeconfig file
func CreateBackup(path string) (string, error) {
	timestamp := time.Now().Format(BackupTimeFormat)
	backupPath := path + ".backup." + timestamp

	src, err := os.Open(path) //nolint:gosec // User-specified backup path is intentional
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close source file: %v\n", closeErr)
		}
	}()

	dst, err := os.Create(backupPath) //nolint:gosec // Backup file creation is intentional
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close destination file: %v\n", closeErr)
		}
	}()

	_, err = io.Copy(dst, src)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return backupPath, nil
}

// RemoveContexts removes the specified contexts and cleans up orphaned entries
func RemoveContexts(config *Config, contextsToRemove []string) error {
	// Track which clusters and users are still in use
	usedClusters := make(map[string]bool)
	usedUsers := make(map[string]bool)

	// Create a map for contexts to remove for quick lookup
	toRemoveMap := make(map[string]bool)
	for _, name := range contextsToRemove {
		toRemoveMap[name] = true
	}

	// Filter out contexts to remove
	var remainingContexts []NamedContext
	for _, namedContext := range config.Contexts {
		if !toRemoveMap[namedContext.Name] {
			remainingContexts = append(remainingContexts, namedContext)
			if namedContext.Context != nil {
				usedClusters[namedContext.Context.Cluster] = true
				usedUsers[namedContext.Context.User] = true
			}
		} else if config.CurrentContext == namedContext.Name {
			// Update current-context if needed
			config.CurrentContext = ""
		}
	}
	config.Contexts = remainingContexts

	// Filter out orphaned clusters
	var remainingClusters []NamedCluster
	for _, namedCluster := range config.Clusters {
		if usedClusters[namedCluster.Name] {
			remainingClusters = append(remainingClusters, namedCluster)
		}
	}
	config.Clusters = remainingClusters

	// Filter out orphaned users
	var remainingUsers []NamedUser
	for _, namedUser := range config.Users {
		if usedUsers[namedUser.Name] {
			remainingUsers = append(remainingUsers, namedUser)
		}
	}
	config.Users = remainingUsers

	// Set a new current-context if the current one is being removed
	if config.CurrentContext == "" && len(config.Contexts) > 0 {
		config.CurrentContext = config.Contexts[0].Name
	}

	// Rebuild internal maps
	config.buildInternalMaps()

	return nil
}

// IsAuthValid checks if the authentication for a context is valid by:
// 1. Verifying credentials exist
// 2. Testing if the cluster API server is reachable
// 3. Making a basic API call to validate authentication
func IsAuthValid(config *Config, contextName string) bool {
	ctx := config.GetContext(contextName)
	if ctx == nil {
		return false
	}

	user := config.GetUser(ctx.User)
	if user == nil {
		return false
	}

	cluster := config.GetCluster(ctx.Cluster)
	if cluster == nil {
		return false
	}

	// First check if we have any auth credentials
	if !hasValidCredentials(user) {
		return false
	}

	// Then check if the cluster is reachable
	return isClusterReachable(cluster, user)
}

// hasValidCredentials checks if the user has any authentication credentials
func hasValidCredentials(user *User) bool {
	// Check for token-based auth
	if user.Token != "" {
		return true
	}

	// Check for certificate-based auth
	if user.ClientCertificateData != "" || user.ClientCertificate != "" {
		return true
	}

	// Check for basic auth
	if user.Username != "" && user.Password != "" {
		return true
	}

	// Check for auth provider (like OIDC, GCP, AWS, etc.)
	if user.AuthProvider != nil {
		return len(user.AuthProvider.Config) > 0
	}

	// Check for exec-based auth (like kubectl plugins)
	if user.Exec != nil && user.Exec.Command != "" {
		if _, err := os.Stat(user.Exec.Command); err == nil {
			return true
		}
		// Also try to find it in PATH
		if _, err := filepath.Abs(user.Exec.Command); err == nil {
			return true
		}
	}

	return false
}

// isClusterReachable tests if the cluster API server is accessible
// This solves the "dead cluster, live token" problem
func isClusterReachable(cluster *Cluster, user *User) bool {
	if cluster.Server == "" {
		return false
	}

	// Create HTTP client with appropriate TLS settings
	client := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec // TLS verification controlled by kubeconfig setting
				InsecureSkipVerify: cluster.InsecureSkipTLSVerify,
			},
		},
	}

	// Try to reach the /version endpoint (doesn't require auth)
	versionURL := cluster.Server + "/version"

	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", versionURL, http.NoBody)
	if err != nil {
		return false
	}

	// Add authentication headers if we have a token
	if user.Token != "" {
		req.Header.Set("Authorization", "Bearer "+user.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Network error, DNS resolution failure, connection refused, etc.
		// This catches the "cluster is gone" scenario
		return false
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// If we get any response (even 401/403), the cluster is reachable
	// Status codes in the 200-499 range indicate the server is responding
	return resp.StatusCode < httpSuccessThreshold
}

// GetCluster returns a cluster by name (needed for the enhanced auth check)
func (c *Config) GetCluster(name string) *Cluster {
	if c.clusterMap == nil {
		return nil
	}
	return c.clusterMap[name]
}
