package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServersConfig(t *testing.T) {
	// Create a temporary directory for test config files
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Save original working directory and change to tmpDir
	// because ValidateServersConfig searches in ./configs and .
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name          string
		serversYaml   string
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid configuration",
			serversYaml: `
nginx_servers:
  - group: "prod"
    servers:
      - name: "web-01"
        host: "192.168.1.1"
        port: 22
        user: "root"
        auth:
          method: "ssh"
          key_path: "/root/.ssh/id_rsa"
        nginx_config_dir: "/etc/nginx"
`,
			expectError: false,
		},
		{
			name: "Empty configuration",
			serversYaml: `
nginx_servers: []
`,
			expectError:   true,
			errorContains: "no nginx server groups defined",
		},
		{
			name: "Missing group name",
			serversYaml: `
nginx_servers:
  - group: ""
    servers:
      - name: "web-01"
        host: "192.168.1.1"
        port: 22
        user: "root"
        auth:
          method: "ssh"
        nginx_config_dir: "/etc/nginx"
`,
			expectError:   true,
			errorContains: "found group with empty name",
		},
		{
			name: "Group with no servers",
			serversYaml: `
nginx_servers:
  - group: "empty-group"
    servers: []
`,
			expectError:   true,
			errorContains: "has no servers defined",
		},
		{
			name: "Invalid IP address",
			serversYaml: `
nginx_servers:
  - group: "prod"
    servers:
      - name: "web-01"
        host: "invalid-ip"
        port: 22
        user: "root"
        auth:
          method: "ssh"
        nginx_config_dir: "/etc/nginx"
`,
			expectError:   true,
			errorContains: "not a valid IP address",
		},
		{
			name: "Invalid port",
			serversYaml: `
nginx_servers:
  - group: "prod"
    servers:
      - name: "web-01"
        host: "192.168.1.1"
        port: 70000
        user: "root"
        auth:
          method: "ssh"
        nginx_config_dir: "/etc/nginx"
`,
			expectError:   true,
			errorContains: "port 70000 is invalid",
		},
		{
			name: "Missing user",
			serversYaml: `
nginx_servers:
  - group: "prod"
    servers:
      - name: "web-01"
        host: "192.168.1.1"
        port: 22
        user: ""
        auth:
          method: "ssh"
        nginx_config_dir: "/etc/nginx"
`,
			expectError:   true,
			errorContains: "user is empty",
		},
		{
			name: "Missing auth method",
			serversYaml: `
nginx_servers:
  - group: "prod"
    servers:
      - name: "web-01"
        host: "192.168.1.1"
        port: 22
        user: "root"
        auth:
          method: ""
        nginx_config_dir: "/etc/nginx"
`,
			expectError:   true,
			errorContains: "auth method is not set",
		},
		{
			name: "Missing nginx_config_dir",
			serversYaml: `
nginx_servers:
  - group: "prod"
    servers:
      - name: "web-01"
        host: "192.168.1.1"
        port: 22
        user: "root"
        auth:
          method: "ssh"
        nginx_config_dir: ""
`,
			expectError:   true,
			errorContains: "nginx_config_dir is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write the servers.yaml file
			err := os.WriteFile(filepath.Join(tmpDir, "servers.yaml"), []byte(tt.serversYaml), 0644)
			require.NoError(t, err)

			groups, err := ValidateServersConfig()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, groups)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, groups)
			}
		})
	}
}
