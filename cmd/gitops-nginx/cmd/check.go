package cmd

import (
	"fmt"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check servers.yaml configuration",
	Long:  `Validate the servers.yaml configuration file for syntax errors and required fields.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverGroups, err := config.ValidateServersConfig()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		fmt.Println("Configuration validation results:")
		fmt.Println("================================")

		totalServers := 0
		for _, group := range serverGroups {
			fmt.Printf("\nGroup: %s\n", group.Group)
			for _, server := range group.Servers {
				totalServers++
				fmt.Printf("  - %s (%s:%d)\n", server.Name, server.Host, server.Port)
			}
		}

		fmt.Printf("\n================================\n")
		fmt.Printf("Total: %d groups, %d servers\n", len(serverGroups), totalServers)
		fmt.Println("Configuration check completed successfully!")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
