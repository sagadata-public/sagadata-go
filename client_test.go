package genesiscloud

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewGenesisCloudClient_StaticToken(t *testing.T) {
	receivedToken := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Auth-Token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ssh_keys":[]}`))
	}))
	defer server.Close()

	client, err := NewGenesisCloudClient(ClientConfig{
		Endpoint: server.URL,
		Token:    "static-test-token",
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}

	if receivedToken != "static-test-token" {
		t.Errorf("expected token %q, got %q", "static-test-token", receivedToken)
	}
}

func TestNewGenesisCloudClient_TokenFile(t *testing.T) {
	receivedToken := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Auth-Token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ssh_keys":[]}`))
	}))
	defer server.Close()

	// Create a temporary token file
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	if err := os.WriteFile(tokenFile, []byte("file-based-token\n"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	client, err := NewGenesisCloudClient(ClientConfig{
		Endpoint:  server.URL,
		TokenFile: tokenFile,
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}

	if receivedToken != "file-based-token" {
		t.Errorf("expected token %q, got %q", "file-based-token", receivedToken)
	}
}

func TestNewGenesisCloudClient_TokenFileDynamicUpdate(t *testing.T) {
	receivedTokens := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTokens = append(receivedTokens, r.Header.Get("X-Auth-Token"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ssh_keys":[]}`))
	}))
	defer server.Close()

	// Create a temporary token file
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token")
	if err := os.WriteFile(tokenFile, []byte("initial-token"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	client, err := NewGenesisCloudClient(ClientConfig{
		Endpoint:  server.URL,
		TokenFile: tokenFile,
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	// First request with initial token
	_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error making first request: %v", err)
	}

	// Update the token file
	if err := os.WriteFile(tokenFile, []byte("updated-token"), 0600); err != nil {
		t.Fatalf("failed to update token file: %v", err)
	}

	// Second request should use updated token
	_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error making second request: %v", err)
	}

	if len(receivedTokens) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(receivedTokens))
	}
	if receivedTokens[0] != "initial-token" {
		t.Errorf("first request: expected token %q, got %q", "initial-token", receivedTokens[0])
	}
	if receivedTokens[1] != "updated-token" {
		t.Errorf("second request: expected token %q, got %q", "updated-token", receivedTokens[1])
	}
}

func TestNewGenesisCloudClient_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		config      ClientConfig
		expectError string
	}{
		{
			name:        "neither token nor token file",
			config:      ClientConfig{Endpoint: "http://localhost"},
			expectError: "either ClientConfig.Token or ClientConfig.TokenFile is required",
		},
		{
			name: "both token and token file",
			config: ClientConfig{
				Endpoint:  "http://localhost",
				Token:     "token",
				TokenFile: "/some/file",
			},
			expectError: "ClientConfig.Token and ClientConfig.TokenFile are mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGenesisCloudClient(tt.config)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.expectError {
				t.Errorf("expected error %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

func TestNewGenesisCloudClient_TokenFileErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("missing file", func(t *testing.T) {
		client, err := NewGenesisCloudClient(ClientConfig{
			Endpoint:  server.URL,
			TokenFile: "/nonexistent/path/to/token",
		})
		if err != nil {
			t.Fatalf("unexpected error creating client: %v", err)
		}

		_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
		if err == nil {
			t.Fatal("expected error for missing token file, got nil")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenFile := filepath.Join(tmpDir, "token")
		if err := os.WriteFile(tokenFile, []byte(""), 0600); err != nil {
			t.Fatalf("failed to write token file: %v", err)
		}

		client, err := NewGenesisCloudClient(ClientConfig{
			Endpoint:  server.URL,
			TokenFile: tokenFile,
		})
		if err != nil {
			t.Fatalf("unexpected error creating client: %v", err)
		}

		_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
		if err == nil {
			t.Fatal("expected error for empty token file, got nil")
		}
	})

	t.Run("whitespace only file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenFile := filepath.Join(tmpDir, "token")
		if err := os.WriteFile(tokenFile, []byte("   \n\t  \n"), 0600); err != nil {
			t.Fatalf("failed to write token file: %v", err)
		}

		client, err := NewGenesisCloudClient(ClientConfig{
			Endpoint:  server.URL,
			TokenFile: tokenFile,
		})
		if err != nil {
			t.Fatalf("unexpected error creating client: %v", err)
		}

		_, err = client.ListSSHKeysPaginatedWithResponse(t.Context(), nil)
		if err == nil {
			t.Fatal("expected error for whitespace-only token file, got nil")
		}
	})
}
