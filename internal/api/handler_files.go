package api

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/diff"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func (s *Server) handleGetTree(c *gin.Context) {
	group := c.Query("group")
	host := c.Query("host")
	mode := c.Query("mode") // "preview" (git) or "prod" (remote in etcd)

	if group == "" || host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group and host are required"})
		return
	}
	if mode != "preview" && mode != "prod" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 'preview', or 'prod'"})
		return
	}

	// Find server config
	srvCfg := s.findServerConfig(group, host)
	if srvCfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	configDirSuffix := filepath.Base(srvCfg.NginxConfigDir)
	gitPrefix := path.Join(s.cfg.Sync.GitSyncer.KeyPrefix, group, host, configDirSuffix)
	previewPrefix := path.Join(s.cfg.Sync.PreviewSyncer.KeyPrefix, group, host, configDirSuffix)
	remotePrefix := path.Join(s.cfg.Sync.NginxSyncer.KeyPrefix, group, host, configDirSuffix)

	// Determine etcd prefix based on mode for paths list
	var err error
	var prefix string
	var gitResp *clientv3.GetResponse
	var previewResp *clientv3.GetResponse
	var remoteResp *clientv3.GetResponse
	var targetHashes map[string]string
	if mode == "preview" {
		prefix = previewPrefix
		previewResp, err = s.etcdClient.GetPrefix(c.Request.Context(), previewPrefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		previewHashes := make(map[string]string)
		for _, kv := range previewResp.Kvs {
			key := string(kv.Key)
			if strings.HasSuffix(key, ".hash") {
				relPath := strings.TrimPrefix(strings.TrimSuffix(key, ".hash"), previewPrefix)
				previewHashes[relPath] = string(kv.Value)
			}
		}

		targetHashes = previewHashes
	} else {
		prefix = gitPrefix
		gitResp, err = s.etcdClient.GetPrefix(c.Request.Context(), gitPrefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		gitHashes := make(map[string]string)
		for _, kv := range gitResp.Kvs {
			key := string(kv.Key)
			if strings.HasSuffix(key, ".hash") {
				relPath := strings.TrimPrefix(strings.TrimSuffix(key, ".hash"), gitPrefix)
				gitHashes[relPath] = string(kv.Value)
			}
		}

		targetHashes = gitHashes
	}

	// Get data from all sources to determine status
	remoteResp, err = s.etcdClient.GetPrefix(c.Request.Context(), remotePrefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	remoteHashes := make(map[string]string)
	for _, kv := range remoteResp.Kvs {
		key := string(kv.Key)
		if strings.HasSuffix(key, ".hash") {
			relPath := strings.TrimPrefix(strings.TrimSuffix(key, ".hash"), remotePrefix)
			remoteHashes[relPath] = string(kv.Value)
		}
	}

	var paths []string
	fileStatuses := make(map[string]string)

	// Determine which hash set to use for status comparison
	// If mode is preview, we compare preview vs remote
	// If mode is prod, we compare prod vs remote
	sourceHashes := remoteHashes

	// Track all seen paths to avoid duplicates
	seenPaths := make(map[string]struct{})

	// Add files from target (preview or git)
	for relPath := range targetHashes {
		trimmedPath := strings.TrimPrefix(relPath, "/")
		paths = append(paths, trimmedPath)
		seenPaths[relPath] = struct{}{}

		targetHash := targetHashes[relPath]
		remoteHash, exists := sourceHashes[relPath]

		if !exists {
			fileStatuses[trimmedPath] = "added"
		} else if targetHash != remoteHash {
			fileStatuses[trimmedPath] = "modified"
		}
	}

	// Add files that exist in remote but not in target (deleted)
	for relPath := range sourceHashes {
		if _, ok := seenPaths[relPath]; !ok {
			trimmedPath := strings.TrimPrefix(relPath, "/")
			paths = append(paths, trimmedPath)
			fileStatuses[trimmedPath] = "deleted"
		}
	}

	c.JSON(http.StatusOK, TreeResponse{
		Prefix:       prefix,
		Paths:        paths,
		FileStatuses: fileStatuses,
	})
}

func (s *Server) handleGetTripleDiff(c *gin.Context) {
	group := c.Query("group")
	host := c.Query("host")
	mode := c.Query("mode")
	relPath := c.Query("path")

	if group == "" || host == "" || relPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group, host and path are required"})
		return
	}
	if mode != "preview" && mode != "prod" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 'preview', or 'prod'"})
		return
	}

	// Find server config
	srvCfg := s.findServerConfig(group, host)
	if srvCfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	configDirSuffix := filepath.Base(srvCfg.NginxConfigDir)

	// Sanitize relPath: remove prefixes if present
	// This handles cases where the frontend sends the full etcd key instead of the relative path
	gitPrefix := path.Join(s.cfg.Sync.GitSyncer.KeyPrefix, group, host, configDirSuffix)
	previewPrefix := path.Join(s.cfg.Sync.PreviewSyncer.KeyPrefix, group, host, configDirSuffix)
	remotePrefix := path.Join(s.cfg.Sync.NginxSyncer.KeyPrefix, group, host, configDirSuffix)

	// strings cut Prefix
	// strings.CutPrefix(relPath, previewPrefix)
	// strings.CutPrefix(relPath, gitPrefix)
	// strings.CutPrefix(relPath, remotePrefix)

	if strings.HasPrefix(relPath, previewPrefix) {
		relPath = strings.TrimPrefix(relPath, previewPrefix)
	} else if strings.HasPrefix(relPath, gitPrefix) {
		relPath = strings.TrimPrefix(relPath, gitPrefix)
	} else if strings.HasPrefix(relPath, remotePrefix) {
		relPath = strings.TrimPrefix(relPath, remotePrefix)
	}
	relPath = strings.TrimPrefix(relPath, "/")

	// Remote content (from etcd)
	remoteKey := path.Join(s.cfg.Sync.NginxSyncer.KeyPrefix, group, host, configDirSuffix, relPath)
	remoteResp, err := s.etcdClient.Get(c.Request.Context(), remoteKey)
	remoteContent := ""
	if err == nil && len(remoteResp.Kvs) > 0 {
		remoteContent = string(remoteResp.Kvs[0].Value)
	}

	// Compare content (from etcd)
	var compareKey string
	var compareLabel string
	if mode == "preview" {
		compareKey = path.Join(s.cfg.Sync.PreviewSyncer.KeyPrefix, group, host, configDirSuffix, relPath)
		compareLabel = "Preview"
	} else {
		compareKey = path.Join(s.cfg.Sync.GitSyncer.KeyPrefix, group, host, configDirSuffix, relPath)
		compareLabel = "Production"
	}

	compareResp, err := s.etcdClient.Get(c.Request.Context(), compareKey)
	compareContent := ""
	if err == nil && len(compareResp.Kvs) > 0 {
		compareContent = string(compareResp.Kvs[0].Value)
	}

	// Generate diff
	diffResult, err := diff.GenerateUnifiedDiff(remoteContent, compareContent, "remote", compareLabel)
	diffText := ""
	if err == nil {
		diffText = diffResult.UnifiedDiff
	}

	c.JSON(http.StatusOK, TripleDiffResponse{
		Path:           relPath,
		RemoteContent:  remoteContent,
		CompareContent: compareContent,
		Diff:           diffText,
		Mode:           mode,
		CompareLabel:   compareLabel,
	})
}

// findServerConfig is a helper function to find a server configuration by group and host.
func (s *Server) findServerConfig(group, host string) *config.ServerConfig {
	for _, g := range s.cfg.NginxServers {
		if g.Group == group {
			for i := range g.Servers {
				if g.Servers[i].Host == host {
					return &g.Servers[i]
				}
			}
		}
	}
	return nil
}
