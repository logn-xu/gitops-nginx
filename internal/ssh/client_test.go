package ssh

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/logn-xu/gitops-nginx/internal/config"
)

func getTestServerConfig() *config.ServerConfig {
	host := os.Getenv("SSH_HOST")
	if host == "" {
		return nil
	}

	port, _ := strconv.Atoi(os.Getenv("SSH_PORT"))
	if port == 0 {
		port = 22
	}

	user := os.Getenv("SSH_USER")
	if user == "" {
		user = "root"
	}

	authMethod := os.Getenv("SSH_AUTH_METHOD")
	if authMethod == "" {
		authMethod = "password"
	}

	password := os.Getenv("SSH_PASSWORD")
	keyPath := os.Getenv("SSH_KEY_PATH")

	return &config.ServerConfig{
		Name: "test-server",
		Host: host,
		Port: port,
		User: user,
		Auth: config.ServerAuthConfig{
			Method:   authMethod,
			Password: password,
			KeyPath:  keyPath,
		},
	}
}

func TestNewClient(t *testing.T) {
	cfg := getTestServerConfig()
	if cfg == nil {
		t.Skip("Skipping SSH integration test (SSH_HOST not set)")
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient returned nil client")
	}
	client.Close()
}

func TestClient_Operations(t *testing.T) {
	cfg := getTestServerConfig()
	if cfg == nil {
		t.Skip("Skipping SSH integration test (SSH_HOST not set)")
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	t.Run("RunCommand", func(t *testing.T) {
		output, err := client.RunCommand("echo 'hello world'")
		if err != nil {
			t.Fatalf("RunCommand failed: %v", err)
		}
		if output != "hello world\n" {
			t.Errorf("Expected 'hello world\\n', got %q", output)
		}
	})

	t.Run("FileOperations", func(t *testing.T) {
		remotePath := "/tmp/gitops-nginx-test-file"
		content := []byte("test content")

		// Test WriteFile
		err := client.WriteFile(remotePath, content)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
		defer client.RunCommand(fmt.Sprintf("rm %s", remotePath))

		// Test ReadFile
		readContent, err := client.ReadFile(remotePath)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(readContent) != string(content) {
			t.Errorf("Expected %q, got %q", string(content), string(readContent))
		}

		// Test GetFileHash
		hash, err := client.GetFileHash(remotePath)
		if err != nil {
			t.Fatalf("GetFileHash failed: %v", err)
		}
		if hash == "" {
			t.Error("Expected non-empty hash")
		}
	})
}
