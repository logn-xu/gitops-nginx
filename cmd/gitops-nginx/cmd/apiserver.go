package cmd

import (
	"github.com/spf13/cobra"
)

var apiserverCmd = &cobra.Command{
	Use:   "apiserver",
	Short: "Start API server without Web UI",
	Long:  `Start the gitops-nginx API server without embedded Web UI. Only API endpoints will be available.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(false)
	},
}

func init() {
	rootCmd.AddCommand(apiserverCmd)
}
