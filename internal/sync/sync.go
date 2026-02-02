package sync

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	gitrepo "github.com/logn-xu/gitops-nginx/internal/git"
	"github.com/logn-xu/gitops-nginx/pkg/log"
)

// Syncer is responsible for syncing Git-based Nginx configurations to etcd.
type Syncer struct {
	etcdClient     *etcd.Client
	serverConfig   *config.ServerConfig
	gitConfig      *config.GitConfig
	groupName      string
	pollInterval   time.Duration
	ignorePatterns []string
	keyPrefix      string
}

// NewSyncer creates a new Syncer.
func NewSyncer(etcdClient *etcd.Client, serverConfig *config.ServerConfig, gitConfig *config.GitConfig, syncConfig *config.SyncConfig, groupName string, pollInterval time.Duration) *Syncer {
	return &Syncer{
		etcdClient:     etcdClient,
		serverConfig:   serverConfig,
		gitConfig:      gitConfig,
		groupName:      groupName,
		pollInterval:   pollInterval,
		ignorePatterns: syncConfig.GitSyncer.IgnorePatterns,
		keyPrefix:      syncConfig.GitSyncer.KeyPrefix,
	}
}

// Reloadable returns true indicating this service can be hot-reloaded
func (s *Syncer) Reloadable() bool { return true }

// Start begins the asynchronous syncing process.
func (s *Syncer) Start(ctx context.Context) error {
	l := log.Logger.WithField("git_syncer", s.serverConfig.Host)
	l.Info("starting git syncer")

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Initial sync
	if err := s.sync(ctx); err != nil {
		l.WithError(err).Error("failed to sync nginx configuration from git")
		return fmt.Errorf("failed to sync nginx configuration from git: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			l.Info("stopping syncer")
			return nil
		case <-ticker.C:
			l.Info("syncing nginx configuration from git to etcd")
			ctx := context.Background()
			if err := s.sync(ctx); err != nil {
				l.WithError(err).Error("failed to sync nginx configuration from git")
				return fmt.Errorf("failed to sync nginx configuration from git: %w", err)
			}
		}
	}
}

// sync syncs the git repository to etcd
func (s *Syncer) sync(ctx context.Context) error {
	l := log.Logger.WithField("git_syncer", s.serverConfig.Host)
	// Check if nginx_config_dir is configured
	if s.serverConfig.NginxConfigDir == "" {
		l.Warn("nginx_config_dir is not configured, skipping sync")
		return nil
	}

	// Sync git repository (clone or pull)
	repo, err := gitrepo.SyncRepository(s.gitConfig)
	if err != nil {
		return fmt.Errorf("failed to sync git repo: %w", err)
	}

	// Get branch
	branchName := s.gitConfig.Branch
	if branchName == "" {
		branchName = "master"
	}

	// Get branch reference
	ref, err := repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		return fmt.Errorf("failed to get branch %s: %w", branchName, err)
	}

	// Get commit object
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit object %s: %w", ref.Hash().String(), err)
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree from commit %s: %w", commit.Hash.String(), err)
	}

	// Get config dir suffix
	configDirSuffix := filepath.Base(s.serverConfig.NginxConfigDir)
	// Get prefix
	prefix := path.Join(s.groupName, s.serverConfig.Host, configDirSuffix)
	// Get etcd prefix
	etcdPrefix := path.Join(s.keyPrefix, s.groupName, s.serverConfig.Host, configDirSuffix)

	// Get all existing keys from etcd for this server to avoid multiple Get calls
	resp, err := s.etcdClient.GetPrefix(ctx, etcdPrefix)
	if err != nil {
		return fmt.Errorf("failed to get existing keys from etcd: %w", err)
	}
	existingData := make(map[string]string)
	for _, kv := range resp.Kvs {
		existingData[string(kv.Key)] = string(kv.Value)
	}

	// Get desired relative paths
	desiredRel := make(map[string]struct{})

	iter := tree.Files()
	for {
		file, err := iter.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to iterate tree files: %w", err)
		}

		filePath := path.Clean(file.Name)
		// Check if file should be ignored
		if IsIgnored(filePath, s.ignorePatterns) {
			continue
		}

		// Check if file is in the correct directory
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}

		relPath := strings.TrimPrefix(filePath, prefix)
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			continue
		}
		desiredRel[relPath] = struct{}{}

		content, err := file.Contents()
		if err != nil {
			l.WithFields(log.Fields{
				"host": s.serverConfig.Host,
				"file": filePath,
			}).WithError(err).Error("failed to read file content from git")
			continue
		}

		hash := md5.Sum([]byte(content))
		hashStr := hex.EncodeToString(hash[:])

		etcdKey := s.constructEtcdKey(relPath, configDirSuffix)
		etcdHashKey := etcdKey + ".hash"
		etcdCommitKey := etcdKey + ".commit"
		etcdMetaKey := etcdKey + ".meta"

		existingHash := existingData[etcdHashKey]
		_, contentExists := existingData[etcdKey]

		// Only skip if BOTH the hash matches AND the content actually exists in etcd
		if existingHash == hashStr && contentExists {
			l.WithFields(log.Fields{
				"host": s.serverConfig.Host,
				"file": etcdKey,
				"hash": hashStr,
			}).Debug("file hash matches and content exists, skipping sync")
			continue
		}

		if _, err = s.etcdClient.Put(ctx, etcdKey, content); err != nil {
			l.WithFields(log.Fields{
				"host": s.serverConfig.Host,
				"file": etcdKey,
			}).WithError(err).Error("failed to put file into etcd")
			continue
		}

		_, _ = s.etcdClient.Put(ctx, etcdHashKey, hashStr)
		_, _ = s.etcdClient.Put(ctx, etcdCommitKey, commit.Hash.String())

		meta := struct {
			Commit  string `json:"commit"`
			Message string `json:"message"`
		}{
			Commit:  commit.Hash.String(),
			Message: strings.TrimSpace(commit.Message),
		}
		if metaBytes, err := json.Marshal(meta); err == nil {
			_, _ = s.etcdClient.Put(ctx, etcdMetaKey, string(metaBytes))
		}

		l.WithFields(log.Fields{
			"host":     s.serverConfig.Host,
			"file":     etcdKey,
			"hash":     hashStr,
			"commit":   commit.Hash.String(),
			"message":  meta.Message,
			"existing": existingHash,
			"re-sync":  !contentExists && existingHash == hashStr,
		}).Info("synced file from git to etcd")
	}

	// Mirror delete etcd prefix
	if err := mirrorDeleteEtcdPrefix(ctx, s.etcdClient, etcdPrefix, desiredRel); err != nil {
		l.WithFields(log.Fields{
			"host":   s.serverConfig.Host,
			"prefix": etcdPrefix,
		}).WithError(err).Warn("failed to mirror delete etcd prefix")
	}

	return nil
}

// constructEtcdKey constructs the etcd key for a file.
// Format: /gitops-nginx/${group}/${host}/${config_dir_suffix}/xxx
func (s *Syncer) constructEtcdKey(relPath, configDirSuffix string) string {
	return path.Join(s.keyPrefix, s.groupName, s.serverConfig.Host, configDirSuffix, relPath)
}
