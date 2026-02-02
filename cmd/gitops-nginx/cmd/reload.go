package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload servers.yaml configuration (hot reload)",
	Long: `Send SIGHUP signal to the running gitops-nginx process to trigger hot reload of servers.yaml.
This allows updating the nginx server list without restarting the entire application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// First validate the new configuration
		fmt.Println("Validating servers.yaml configuration...")
		_, err := config.ValidateServersConfig()
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		fmt.Println("Configuration validation passed.")

		// Read PID file
		pidFile := "/tmp/gitops-nginx.pid"
		pidData, err := os.ReadFile(pidFile)
		if err != nil {
			return fmt.Errorf("failed to read PID file (%s): %w\nIs gitops-nginx running?", pidFile, err)
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err != nil {
			return fmt.Errorf("invalid PID in file: %w", err)
		}

		// Check if process exists
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process with PID %d: %w", pid, err)
		}

		// Send SIGHUP signal
		fmt.Printf("Sending SIGHUP to process %d...\n", pid)
		if err := process.Signal(syscall.SIGHUP); err != nil {
			return fmt.Errorf("failed to send SIGHUP signal: %w", err)
		}

		fmt.Println("Reload signal sent successfully.")
		fmt.Println("Check the application logs for reload status.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
