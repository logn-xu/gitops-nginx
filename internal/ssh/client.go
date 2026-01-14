package ssh

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/pkg/log"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH and SFTP client.
type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewClient creates a new SSH and SFTP client.
func NewClient(serverConfig *config.ServerConfig) (*Client, error) {
	var authMethod ssh.AuthMethod
	var err error

	switch serverConfig.Auth.Method {
	case "password":
		if serverConfig.Auth.Password == "" {
			return nil, fmt.Errorf("password authentication method requires a password")
		}
		authMethod = ssh.Password(serverConfig.Auth.Password)
	case "key":
		if serverConfig.Auth.KeyPath == "" {
			return nil, fmt.Errorf("key authentication method requires a key path")
		}
		key, err := os.ReadFile(serverConfig.Auth.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key from %s: %w", serverConfig.Auth.KeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", serverConfig.Auth.Method)
	}

	sshConfig := &ssh.ClientConfig{
		User: serverConfig.User,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		// In a real-world application, you should use a more secure host key callback.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH server at %s: %w", addr, err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		// Close the SSH client if SFTP client creation fails
		sshClient.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	client := &Client{
		sshClient:  sshClient,
		sftpClient: sftpClient,
	}

	// Start keep-alive heartbeat
	go client.keepAlive()

	return client, nil
}

// keepAlive sends a periodic heartbeat to keep the SSH connection alive.
func (c *Client) keepAlive() {
	keepAlive := 30 * time.Second

	for {
		// Send a global request to keep the connection alive
		_, _, err := c.sshClient.SendRequest("keepalive@openssh.com", true, nil)
		if err != nil {
			log.Logger.Error("Send a global request to keep the connection alive error")
			return
		}
		time.Sleep(keepAlive)
	}
}

// Close closes the SSH and SFTP connections.
func (c *Client) Close() error {
	var errs []error
	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close SFTP client: %w", err))
		}
	}
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close SSH client: %w", err))
		}
	}
	return errors.Join(errs...)
}

// RunCommand runs a command on the remote server.
func (c *Client) RunCommand(cmd string) (string, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("failed to run command '%s': %w", cmd, err)
	}
	return string(output), nil
}

// ReadFile reads the content of a remote file using SFTP.
func (c *Client) ReadFile(path string) ([]byte, error) {
	file, err := c.sftpClient.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file %s: %w", path, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote file %s: %w", path, err)
	}
	return content, nil
}

// WriteFile writes data to a remote file using SFTP.
// It creates the file if it doesn't exist, and truncates it if it does.
func (c *Client) WriteFile(path string, data []byte) error {
	file, err := c.sftpClient.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", path, err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to remote file %s: %w", path, err)
	}
	return nil
}

// GetFileHash calculates the MD5 hash of a remote file.
func (c *Client) GetFileHash(path string) (string, error) {
	// Use md5sum command to get file hash
	cmd := fmt.Sprintf("md5sum %s | awk '{print $1}'", path)
	output, err := c.RunCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get file hash for %s: %w", path, err)
	}
	return strings.TrimSpace(output), nil
}
