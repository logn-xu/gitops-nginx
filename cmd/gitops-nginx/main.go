package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/internal/manager"
	"github.com/logn-xu/gitops-nginx/internal/sync"
	"github.com/logn-xu/gitops-nginx/pkg/log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Logger.Fatalf("failed to load configuration: %v", err)
	}

	log.SetLevel(cfg.Log.Level)
	log.Logger.Info("configuration loaded successfully")

	etcdClient, err := etcd.NewClient(cfg.Etcd)
	if err != nil {
		log.Logger.Fatalf("failed to create etcd client: %v", err)
	}
	defer etcdClient.Close()
	log.Logger.Info("etcd client created successfully")

	// Create service manager
	mgr := manager.NewManager()

	// Initialize NginxSyncers
	pollInterval := time.Duration(cfg.Sync.NginxSyncer.IntervalSeconds) * time.Second
	if pollInterval == 0 {
		pollInterval = 60 * time.Second
	}

	for _, group := range cfg.NginxServers {
		for i := range group.Servers {
			// Use pointer to the actual server config in the slice
			server := &group.Servers[i]
			syncer := sync.NewNginxSyncer(etcdClient, server, cfg.Sync, group.Group, pollInterval)
			mgr.Add(syncer)
		}
	}

	// TODO: Add Gin server and other syncers here
	// e.g., mgr.Add(api.NewServer(cfg))

	// Start all services
	log.Logger.Info("starting all services...")
	mgr.Start()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger.Info("shutting down services...")

	mgr.Stop()
	log.Logger.Info("services exited")
}
