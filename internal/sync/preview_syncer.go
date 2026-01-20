package sync

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/pkg/log"
	"github.com/sirupsen/logrus"
)

// PreviewSyncer watches for file changes in Git repository and syncs to preview etcd path
type PreviewSyncer struct {
	etcdClient     *etcd.Client
	serverConfig   *config.ServerConfig
	gitConfig      *config.GitConfig
	groupName      string
	repoPathAbs    string
	watcher        *fsnotify.Watcher
	ignorePatterns []string
	pollInterval   time.Duration
	keyPrefix      string
}

// loadGitignore loads .gitignore patterns from the repository
func loadGitignore(repoPath string) ([]string, error) {
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	return patterns, nil
}

// NewPreviewSyncer creates a new PreviewSyncer
func NewPreviewSyncer(etcdClient *etcd.Client, serverConfig *config.ServerConfig, gitConfig *config.GitConfig, syncConfig *config.SyncConfig, groupName string) (*PreviewSyncer, error) {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	repoPathAbs := gitConfig.RepoPath
	if repoPathAbs != "" {
		if abs, err := filepath.Abs(gitConfig.RepoPath); err == nil {
			repoPathAbs = abs
		}
	}

	// Load .gitignore patterns
	ignorePatterns, err := loadGitignore(gitConfig.RepoPath)
	if err != nil {
		logrus.WithError(err).Warn("failed to load .gitignore, will not filter files")
		ignorePatterns = []string{}
	}

	// Append configured ignore patterns
	ignorePatterns = append(ignorePatterns, syncConfig.PreviewSyncer.IgnorePatterns...)

	return &PreviewSyncer{
		etcdClient:     etcdClient,
		serverConfig:   serverConfig,
		gitConfig:      gitConfig,
		groupName:      groupName,
		repoPathAbs:    repoPathAbs,
		watcher:        watcher,
		ignorePatterns: ignorePatterns,
		pollInterval:   5 * time.Second,
		keyPrefix:      syncConfig.PreviewSyncer.KeyPrefix,
	}, nil
}

// Start begins the file watching process
func (ps *PreviewSyncer) Start(ctx context.Context) error {
	log := log.Logger.WithField("preview_syncer", ps.serverConfig.Host)
	log.Info("starting preview syncer")

	// Get the absolute path of the git repository
	repoPath := ps.repoPathAbs
	if repoPath == "" {
		return fmt.Errorf("git repo path is not configured")
	}
	if abs, err := filepath.Abs(repoPath); err == nil {
		repoPath = abs
		ps.repoPathAbs = abs
	}

	// Initial sync of existing files
	if err := ps.initialSync(ctx); err != nil {
		log.WithError(err).Error("failed to perform initial sync")
	}

	// Watch the repository recursively for changes.
	if err := ps.addWatchRecursive(repoPath); err != nil {
		return fmt.Errorf("failed to watch git repository: %w", err)
	}

	log.WithField("path", repoPath).Info("watching git repository for changes")
	log.WithField("path", repoPath).Info("starting preview syncer periodic scan")

	go func() {
		defer ps.watcher.Close()
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping preview syncer")
				return
			case event, ok := <-ps.watcher.Events:
				if !ok {
					return
				}
				ps.handleFileEvent(ctx, event)
			case err, ok := <-ps.watcher.Errors:
				if !ok {
					return
				}
				log.WithError(err).Error("file watcher error")
			}
		}
	}()

	go ps.pollSyncLoop(ctx)

	return nil
}

// initialSync performs initial sync of all existing files in the repository
func (ps *PreviewSyncer) initialSync(ctx context.Context) error {
	l := log.Logger.WithField("preview_syncer", ps.serverConfig.Host)
	l.Info("performing initial sync of existing files")

	repoPath := ps.repoPathAbs
	if repoPath == "" {
		return fmt.Errorf("git repo path is not configured")
	}

	configDirSuffix := filepath.Base(ps.serverConfig.NginxConfigDir)
	expectedPrefix := path.Join(ps.groupName, ps.serverConfig.Host, configDirSuffix)
	etcdPrefix := path.Join(ps.keyPrefix, ps.groupName, ps.serverConfig.Host, configDirSuffix)

	// Get all existing keys from etcd to avoid multiple Get calls
	resp, err := ps.etcdClient.GetPrefix(ctx, etcdPrefix)
	if err != nil {
		return fmt.Errorf("failed to get existing keys from etcd: %w", err)
	}
	existingData := make(map[string]string)
	for _, kv := range resp.Kvs {
		existingData[string(kv.Key)] = string(kv.Value)
	}

	desiredRel := make(map[string]struct{})

	// Walk through the repository directory and sync relevant files
	err = filepath.WalkDir(repoPath, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Skip ignored files
		if IsIgnored(filePath, ps.ignorePatterns) {
			return nil
		}

		// Convert to relative path from repo root
		rel, err := filepath.Rel(repoPath, filePath)
		if err != nil {
			return nil
		}
		relPath := filepath.ToSlash(rel)

		// Check if file matches the expected nginx config pattern
		if !strings.HasPrefix(relPath, expectedPrefix) {
			return nil
		}

		// Extract the file path relative to the config directory
		fileRelPath := strings.TrimPrefix(relPath, expectedPrefix)
		fileRelPath = strings.TrimPrefix(fileRelPath, "/")
		if fileRelPath == "" {
			return nil
		}
		desiredRel[fileRelPath] = struct{}{}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			l.WithError(err).Errorf("failed to read file %s", filePath)
			return nil
		}

		hash := md5.Sum(content)
		hashStr := hex.EncodeToString(hash[:])

		etcdKey := ps.constructEtcdKey(fileRelPath, configDirSuffix)
		etcdHashKey := etcdKey + ".hash"

		existingHash := existingData[etcdHashKey]
		_, contentExists := existingData[etcdKey]

		// Only skip if BOTH the hash matches AND the content actually exists in etcd
		if existingHash == hashStr && contentExists {
			return nil
		}

		if _, err = ps.etcdClient.Put(ctx, etcdKey, string(content)); err != nil {
			l.WithError(err).Errorf("failed to put file %s into etcd", etcdKey)
			return nil
		}

		_, _ = ps.etcdClient.Put(ctx, etcdHashKey, hashStr)

		l.WithFields(log.Fields{
			"host":     ps.serverConfig.Host,
			"file":     etcdKey,
			"hash":     hashStr,
			"existing": existingHash,
			"re-sync":  !contentExists && existingHash == hashStr,
		}).Info("synced file to preview etcd")

		return nil
	})
	if err != nil {
		return err
	}

	if err := mirrorDeleteEtcdPrefix(ctx, ps.etcdClient, etcdPrefix, desiredRel); err != nil {
		l.WithFields(log.Fields{
			"prefix": etcdPrefix,
		}).WithError(err).Warn("failed to mirror delete etcd prefix")
	}

	return nil
}

// constructEtcdKey constructs the etcd key for a file.
func (ps *PreviewSyncer) constructEtcdKey(relPath, configDirSuffix string) string {
	return path.Join(ps.keyPrefix, ps.groupName, ps.serverConfig.Host, configDirSuffix, relPath)
}

// addWatchRecursive adds a recursive watch on the directory
func (ps *PreviewSyncer) addWatchRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if IsIgnored(path, ps.ignorePatterns) {
				return filepath.SkipDir
			}
			return ps.watcher.Add(path)
		}
		return nil
	})
}

// handleFileEvent handles a single fsnotify event
func (ps *PreviewSyncer) handleFileEvent(ctx context.Context, event fsnotify.Event) {
	l := log.Logger.WithFields(log.Fields{
		"preview_syncer": ps.serverConfig.Host,
		"event":          event.Op.String(),
		"file":           event.Name,
	})

	if IsIgnored(event.Name, ps.ignorePatterns) {
		return
	}

	// For simplicity, any change triggers an initialSync to ensure consistency.
	// In a high-volume environment, this should be throttled or more targeted.
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
		l.Debug("file change detected, triggering sync")
		if err := ps.initialSync(ctx); err != nil {
			l.WithError(err).Error("failed to sync after file event")
		}
	}
}

// pollSyncLoop periodically scans the repository to ensure everything is in sync
func (ps *PreviewSyncer) pollSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(ps.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ps.initialSync(ctx); err != nil {
				log.Logger.WithField("preview_syncer", ps.serverConfig.Host).WithError(err).Error("periodic sync failed")
			}
		}
	}
}
