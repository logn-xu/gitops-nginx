package cmd

import (
	"embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

// PidFilePath returns the path to the PID file for the current user
func PidFilePath() string {
	runDir := fmt.Sprintf("/run/user/%d", os.Getuid())
	if _, err := os.Stat(runDir); os.IsNotExist(err) {
		runDir = os.TempDir()
	}
	return filepath.Join(runDir, "gitops-nginx.pid")
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

	// Add API server (not reloadable)
	if withUI {
		mgr.Add(api.NewServer(cfg, etcdClient, dist))
	} else {
		mgr.Add(api.NewServerWithoutUI(cfg, etcdClient))
	}

	// Create syncer factory for reload
	createSyncers := func() []manager.Service {
		serverGroups, err := config.ValidateServersConfig()
		if err != nil {
			log.Logger.WithError(err).Error("failed to reload servers config")
			return nil
		}

		gitInterval := max(time.Duration(cfg.Sync.GitSyncer.IntervalSeconds)*time.Second, 15*time.Second)
		nginxInterval := max(time.Duration(cfg.Sync.NginxSyncer.IntervalSeconds)*time.Second, 15*time.Second)

		var services []manager.Service
		for _, group := range serverGroups {
			for _, server := range group.Servers {
				nginxSyncer := sync.NewNginxSyncer(etcdClient, &server, &cfg.Sync, group.Group, nginxInterval)
				gitSyncer := sync.NewSyncer(etcdClient, &server, &cfg.Git, &cfg.Sync, group.Group, gitInterval)
				if previewSyncer, err := sync.NewPreviewSyncer(etcdClient, &server, &cfg.Git, &cfg.Sync, group.Group); err != nil {
					log.Logger.WithError(err).Errorf("failed to create preview syncer for %s", server.Host)
				} else {
					services = append(services, previewSyncer)
				}
				services = append(services, nginxSyncer, gitSyncer)
			}
		}
		return services
	}

	// Add initial syncers
	for _, s := range createSyncers() {
		mgr.AddReloadable(s)
	}

	// Write PID file
	pidFile := PidFilePath()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		log.Logger.WithError(err).Warnf("failed to write PID file: %s", pidFile)
	} else {
		log.Logger.Infof("PID file written to %s", pidFile)
	}
	defer os.Remove(pidFile)

	log.Logger.Info("starting all services...")
	mgr.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for sig := range quit {
		if sig == syscall.SIGHUP {
			log.Logger.Info("received SIGHUP, reloading services...")
			mgr.Reload(createSyncers)
			continue
		}
		break
	}
	log.Logger.Info("shutting down services...")

	mgr.Stop()
	log.Logger.Info("services exited")
	return nil
}
