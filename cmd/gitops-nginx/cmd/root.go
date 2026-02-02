package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via -ldflags
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "gitops-nginx",
	Short: "GitOps-based Nginx configuration management tool",
	Long: `gitops-nginx is a GitOps-based tool for managing Nginx configurations.
It syncs configurations from Git repositories to remote Nginx servers via etcd.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("gitops-nginx version %s\nBuild time: %s\nGit commit: %s\n", Version, BuildTime, GitCommit))
}
