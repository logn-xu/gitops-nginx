package cmd

import (
	"embed"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logn-xu/gitops-nginx/internal/api"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/internal/manager"
	"github.com/logn-xu/gitops-nginx/internal/sync"
	"github.com/logn-xu/gitops-nginx/pkg/log"
	"github.com/spf13/cobra"
)

var dist embed.FS

// SetDist sets the embedded filesystem for the UI
func SetDist(fs embed.FS) {
	dist = fs
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start API server with Web UI",
	Long:  `Start the gitops-nginx API server with embedded Web UI for managing Nginx configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(true)
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runServer(withUI bool) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	log.InitLoggers(&cfg.Log)
	log.Logger.Info("configuration loaded successfully")

	etcdClient, err := etcd.NewClient(cfg.Etcd)
	if err != nil {
		return err
	}
	defer etcdClient.Close()
	log.Logger.Info("etcd client created successfully")

	mgr := manager.NewManager()

	gitPollInterval := time.Duration(cfg.Sync.GitSyncer.IntervalSeconds) * time.Second
	if gitPollInterval <= 0 {
		gitPollInterval = 15 * time.Second
	}

	nginxPollInterval := time.Duration(cfg.Sync.NginxSyncer.IntervalSeconds) * time.Second
	if nginxPollInterval <= 0 {
		nginxPollInterval = 15 * time.Second
	}

	for _, group := range cfg.NginxServers {
		for i := range group.Servers {
			server := &group.Servers[i]
			nginxSyncer := sync.NewNginxSyncer(etcdClient, server, &cfg.Sync, group.Group, nginxPollInterval)
			gitSyncer := sync.NewSyncer(etcdClient, server, &cfg.Git, &cfg.Sync, group.Group, gitPollInterval)
			previewSyncer, err := sync.NewPreviewSyncer(etcdClient, server, &cfg.Git, &cfg.Sync, group.Group)
			if err != nil {
				log.Logger.WithError(err).Errorf("failed to create preview syncer for %s", server.Host)
			} else {
				mgr.Add(previewSyncer)
			}
			mgr.Add(nginxSyncer)
			mgr.Add(gitSyncer)
		}
	}

	if withUI {
		mgr.Add(api.NewServer(cfg, etcdClient, dist))
	} else {
		mgr.Add(api.NewServerWithoutUI(cfg, etcdClient))
	}

	log.Logger.Info("starting all services...")
	mgr.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger.Info("shutting down services...")

	mgr.Stop()
	log.Logger.Info("services exited")
	return nil
}
