package sync

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/internal/ssh"
	"github.com/logn-xu/gitops-nginx/pkg/log"
	"github.com/sirupsen/logrus"
)

// NginxSyncer syncs nginx configuration from remote server to etcd
type NginxSyncer struct {
	etcdClient     *etcd.Client
	serverConfig   *config.ServerConfig
	groupName      string
	pollInterval   time.Duration
	ignorePatterns []string
	keyPrefix      string
}

// NewNginxSyncer creates a new NginxSyncer
func NewNginxSyncer(etcdClient *etcd.Client, serverConfig *config.ServerConfig, syncConfig config.SyncConfig, groupName string, pollInterval time.Duration) *NginxSyncer {
	return &NginxSyncer{
		etcdClient:     etcdClient,
		serverConfig:   serverConfig,
		groupName:      groupName,
		pollInterval:   pollInterval,
		ignorePatterns: syncConfig.NginxSyncer.IgnorePatterns,
		keyPrefix:      syncConfig.NginxSyncer.KeyPrefix,
	}
}

// Start begins the nginx configuration syncing process
func (ns *NginxSyncer) Start(ctx context.Context) error {
	log := log.Logger.WithField("nginx_syncer", ns.serverConfig.Host)
	log.Info("starting nginx syncer")

	ticker := time.NewTicker(ns.pollInterval)
	defer ticker.Stop()

	// Initial sync
	if err := ns.sync(ctx); err != nil {
		log.WithError(err).Error("failed to sync nginx configuration from remote server")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping nginx syncer")
			return ctx.Err()
		case <-ticker.C:
			log.Info("syncing nginx configuration from remote server to etcd")
			if err := ns.sync(ctx); err != nil {
				log.WithError(err).Error("failed to sync nginx configuration from remote server")
			}
		}
	}
}

func (ns *NginxSyncer) sync(ctx context.Context) error {
	log := log.Logger.WithField("nginx_syncer", ns.serverConfig.Host)
	// Check if nginx_config_dir is configured
	if ns.serverConfig.NginxConfigDir == "" {
		log.WithField("nginx_syncer", ns.serverConfig.Host).Warn("nginx_config_dir is not configured, skipping sync")
		return nil
	}

	// TODO: 修改为使用session
	// Connect to remote server via SSH
	sshClient, err := ssh.NewClient(ns.serverConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", ns.serverConfig.Host, err)
	}
	defer sshClient.Close()

	configDirSuffix := filepath.Base(ns.serverConfig.NginxConfigDir)

	// Read nginx configuration files from remote server
	configFiles, err := ns.readRemoteConfigFiles(sshClient, ns.serverConfig.NginxConfigDir)
	if err != nil {
		return fmt.Errorf("failed to read remote config files: %w", err)
	}

	log.WithFields(logrus.Fields{
		"configPath": ns.serverConfig.NginxConfigDir,
		"fileCount":  len(configFiles),
	}).Info("found remote config files")

	// Sync each file to etcd
	desiredRel := make(map[string]struct{}, len(configFiles))
	etcdPrefix := strings.Join([]string{ns.keyPrefix, ns.groupName, ns.serverConfig.Host, configDirSuffix}, "/")
	for _, filePath := range configFiles {
		relPath := strings.TrimPrefix(filePath, "/")
		if relPath == "" {
			continue
		}
		desiredRel[relPath] = struct{}{}

		fullRemotePath := filepath.Join(ns.serverConfig.NginxConfigDir, filePath)
		log.WithFields(logrus.Fields{
			"filePath": filePath,
			"relPath":  relPath,
			"fullPath": fullRemotePath,
		}).Debug("processing remote file")

		// TODO: 优化使用session
		content, err := sshClient.ReadFile(fullRemotePath)
		if err != nil {
			log.WithFields(logrus.Fields{
				"file": filePath,
			}).WithError(err).Error("failed to read remote file")
			continue
		}

		hash := md5.Sum([]byte(content))
		hashStr := hex.EncodeToString(hash[:])

		etcdKey := ns.constructEtcdKey(relPath, configDirSuffix)
		etcdHashKey := etcdKey + ".hash"
		etcdMetaKey := etcdKey + ".meta"

		etcdHashResp, err := ns.etcdClient.Get(ctx, etcdHashKey)
		if err != nil {
			log.WithFields(logrus.Fields{
				"file": etcdKey,
			}).WithError(err).Debug("failed to get etcd hash, will sync")
		}

		existingHash := ""
		if etcdHashResp != nil && len(etcdHashResp.Kvs) > 0 {
			existingHash = string(etcdHashResp.Kvs[0].Value)
		}

		if existingHash == hashStr {
			log.WithFields(logrus.Fields{
				"file": etcdKey,
				"hash": hashStr,
			}).Debug("file hash matches, skipping sync")
			continue
		}

		if _, err = ns.etcdClient.Put(ctx, etcdKey, string(content)); err != nil {
			log.WithFields(logrus.Fields{
				"file": etcdKey,
			}).WithError(err).Error("failed to put file into etcd")
			continue
		}

		_, _ = ns.etcdClient.Put(ctx, etcdHashKey, hashStr)

		meta := struct {
			Source      string    `json:"source"`
			LastUpdated time.Time `json:"last_updated"`
		}{
			Source:      "nginx-remote",
			LastUpdated: time.Now(),
		}
		if metaBytes, err := json.Marshal(meta); err == nil {
			_, _ = ns.etcdClient.Put(ctx, etcdMetaKey, string(metaBytes))
		}

		log.WithFields(logrus.Fields{
			"file":     etcdKey,
			"hash":     hashStr,
			"existing": existingHash,
		}).Info("synced nginx file from remote server to etcd")
	}

	if err := mirrorDeleteEtcdPrefix(ctx, ns.etcdClient, etcdPrefix, desiredRel); err != nil {
		logrus.WithFields(logrus.Fields{
			"nginx_syncer": ns.serverConfig.Name,
			"prefix":       etcdPrefix,
		}).WithError(err).Warn("failed to mirror delete etcd prefix")
	}

	return nil
}

// readRemoteConfigFiles reads configuration files from remote nginx server
func (ns *NginxSyncer) readRemoteConfigFiles(sshClient *ssh.Client, configPath string) ([]string, error) {
	log := log.Logger.WithField("nginx_syncer", ns.serverConfig.Host)
	// List all files recursively in the nginx configuration directory
	//TODO:
	output, err := sshClient.RunCommand(fmt.Sprintf("find %s -type f", configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to list remote config files: %w", err)
	}

	log.WithFields(logrus.Fields{
		"configPath": configPath,
		"output":     output,
	}).Debug("find command output")

	// Parse the find command output to get file paths
	var files []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Extract just the filename from the full path
			relativePath := strings.TrimPrefix(line, configPath+"/")

			if IsIgnored(relativePath, ns.ignorePatterns) {
				log.WithFields(logrus.Fields{
					"file": relativePath,
				}).Debug("ignoring remote file")
				continue
			}

			files = append(files, relativePath)
			log.WithFields(logrus.Fields{
				"fullPath":     line,
				"relativePath": relativePath,
			}).Debug("found remote file")
		}
	}

	log.WithFields(logrus.Fields{
		"fileCount": len(files),
	}).Info("parsed remote config files")

	return files, nil
}

// constructEtcdKey constructs the etcd key for a file.
// Format: /gitops-nginx-remote/${group}/${host}/${config_dir_suffix}/xxx
func (ns *NginxSyncer) constructEtcdKey(relPath, configDirSuffix string) string {
	return strings.Join([]string{ns.keyPrefix, ns.groupName, ns.serverConfig.Host, configDirSuffix, relPath}, "/")
}
