package kubeconfig

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsClusterReachable(t *testing.T) {
	tests := []struct {
		server   func() string
		user     *User
		name     string
		expected bool
	}{
		{
			name: "reachable server with token",
			server: func() string {
				// Create a test server that responds to /version
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/version" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"major":"1","minor":"24"}`))
					}
				}))
				return server.URL
			},
			user: &User{
				Token: "valid-token",
			},
			expected: true,
		},
		{
			name: "unreachable server",
			server: func() string {
				return "https://definitely-does-not-exist.invalid:443"
			},
			user: &User{
				Token: "valid-token",
			},
			expected: false,
		},
		{
			name: "server responds with error but is reachable",
			server: func() string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"unauthorized"}`))
				}))
				return server.URL
			},
			user: &User{
				Token: "invalid-token",
			},
			expected: true, // Server is reachable even if auth fails
		},
		{
			name: "server with 500 error",
			server: func() string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				return server.URL
			},
			user: &User{
				Token: "token",
			},
			expected: false, // 5xx errors indicate server issues
		},
		{
			name: "empty server URL",
			server: func() string {
				return ""
			},
			user: &User{
				Token: "token",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverURL := tt.server()

			cluster := &Cluster{
				Server: serverURL,
			}

			result := isClusterReachable(cluster, tt.user)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for server %s", tt.expected, result, serverURL)
			}
		})
	}
}

func TestHasValidCredentials(t *testing.T) {
	tests := []struct {
		user     *User
		name     string
		expected bool
	}{
		{
			name: "valid token",
			user: &User{
				Token: "some-token",
			},
			expected: true,
		},
		{
			name: "valid certificate data",
			user: &User{
				ClientCertificateData: "cert-data",
			},
			expected: true,
		},
		{
			name: "valid basic auth",
			user: &User{
				Username: "admin",
				Password: "secret",
			},
			expected: true,
		},
		{
			name: "valid auth provider",
			user: &User{
				AuthProvider: &AuthProvider{
					Name: "oidc",
					Config: map[string]string{
						"client-id": "my-client",
					},
				},
			},
			expected: true,
		},
		{
			name:     "no credentials",
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
		{
			name: "auth provider without config",
			user: &User{
				AuthProvider: &AuthProvider{
					Name: "oidc",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasValidCredentials(tt.user)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for user %+v", tt.expected, result, tt.user)
			}
		})
	}
}

func TestGetCluster(t *testing.T) {
	config := &Config{
		Clusters: []NamedCluster{
			{
				Name: "test-cluster",
				Cluster: &Cluster{
					Server: "https://test.com",
				},
			},
		},
	}
	config.buildInternalMaps()

	// Test existing cluster
	cluster := config.GetCluster("test-cluster")
	if cluster == nil {
		t.Error("Expected to find cluster, got nil")
	}
	if cluster.Server != "https://test.com" {
		t.Errorf("Expected server 'https://test.com', got %s", cluster.Server)
	}

	// Test non-existing cluster
	cluster = config.GetCluster("non-existent")
	if cluster != nil {
		t.Errorf("Expected nil for non-existent cluster, got %v", cluster)
	}
}

func TestEnhancedIsAuthValid(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"major":"1","minor":"24"}`))
		}
	}))
	defer server.Close()

	config := &Config{
		Contexts: []NamedContext{
			{
				Name: "reachable-context",
				Context: &Context{
					Cluster: "reachable-cluster",
					User:    "valid-user",
				},
			},
			{
				Name: "unreachable-context",
				Context: &Context{
					Cluster: "unreachable-cluster",
					User:    "valid-user",
				},
			},
		},
		Clusters: []NamedCluster{
			{
				Name: "reachable-cluster",
				Cluster: &Cluster{
					Server: server.URL,
				},
			},
			{
				Name: "unreachable-cluster",
				Cluster: &Cluster{
					Server: "https://does-not-exist.invalid:443",
				},
			},
		},
		Users: []NamedUser{
			{
				Name: "valid-user",
				User: &User{
					Token: "valid-token",
				},
			},
		},
	}
	config.buildInternalMaps()

	// Test reachable cluster
	if !IsAuthValid(config, "reachable-context") {
		t.Error("Expected reachable context to be valid")
	}

	// Test unreachable cluster
	if IsAuthValid(config, "unreachable-context") {
		t.Error("Expected unreachable context to be invalid")
	}

	// Test non-existent context
	if IsAuthValid(config, "non-existent") {
		t.Error("Expected non-existent context to be invalid")
	}
}

// TestReachabilityTimeout ensures we don't hang on slow networks
func TestReachabilityTimeout(t *testing.T) {
	// Create a server that delays response beyond our timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // Longer than our 10s client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cluster := &Cluster{
		Server: server.URL,
	}
	user := &User{
		Token: "token",
	}

	start := time.Now()
	result := isClusterReachable(cluster, user)
	duration := time.Since(start)

	// Should return false due to timeout
	if result {
		t.Error("Expected timeout to result in false")
	}

	// Should complete within reasonable time (our timeout + some buffer)
	if duration > 12*time.Second {
		t.Errorf("Expected timeout around 10s, took %v", duration)
	}
}
