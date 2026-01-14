package main

import (
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/pkg/log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Logger.Fatalf("failed to load configuration: %v", err)
	}

	log.SetLevel(cfg.Log.Level)
	log.Logger.Info("configuration loaded successfully")
}
