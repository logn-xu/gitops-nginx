package api

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/logn-xu/gitops-nginx/internal/ssh"
)

func (s *Server) handleCheckConfig(c *gin.Context) {
	var req CheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mode := c.Query("mode")
	if mode != "preview" && mode != "prod" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 'preview', or 'prod'"})
		return
	}

	// Find server config
	srvCfg := s.findServerConfig(req.Group, req.Server)
	if srvCfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	// 1. Determine etcd prefix
	configDirSuffix := filepath.Base(srvCfg.NginxConfigDir)
	var etcdPrefix string
	if mode == "preview" {
		etcdPrefix = path.Join(s.cfg.Sync.PreviewSyncer.KeyPrefix, req.Group, req.Server, configDirSuffix)
	} else {
		etcdPrefix = path.Join(s.cfg.Sync.GitSyncer.KeyPrefix, req.Group, req.Server, configDirSuffix)
	}

	// 2. Get pool and sync files to remote check directory
	pool, err := s.getPool(srvCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get SSH pool: %v", err)})
		return
	}

	remoteCheckDir := srvCfg.CheckDir
	if remoteCheckDir == "" {
		remoteCheckDir = path.Join(srvCfg.NginxConfigDir, "check")
	}

	scpResult, err := ssh.ScpEtcdToRemote(c.Request.Context(), s.etcdClient, pool, srvCfg, etcdPrefix, remoteCheckDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to sync files to check directory: %v", err)})
		return
	}

	// 3. Run nginx test command using the check directory
	sshClient, err := pool.Get(srvCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get SSH client: %v", err)})
		return
	}
	defer pool.Put(sshClient)

	nginxBinary := srvCfg.NginxBinaryPath
	if nginxBinary == "" {
		nginxBinary = "nginx"
	}

	// Construct nginx -t command with custom config file
	// Assuming nginx.conf is at the root of the synced directory
	mainConfigFile := path.Join(remoteCheckDir, "nginx.conf")
	testCmd := fmt.Sprintf("%s -t -c %s", nginxBinary, mainConfigFile)

	output, err := sshClient.RunCommand(testCmd)
	success := err == nil

	res := CheckResponse{
		OK:   success,
		Mode: mode,
		Sync: &SyncResult{
			Total:        scpResult.Total,
			Skipped:      scpResult.Skipped,
			Added:        scpResult.Added,
			Updated:      scpResult.Updated,
			Deleted:      scpResult.Deleted,
			AddedFiles:   scpResult.AddedFiles,
			UpdatedFiles: scpResult.UpdatedFiles,
			DeletedFiles: scpResult.DeletedFiles,
		},
		Nginx: &NginxExecOutput{
			Command: testCmd,
			OK:      success,
			Output:  output,
		},
	}

	c.JSON(http.StatusOK, res)
}

func (s *Server) handleUpdatePrepare(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mode := c.Query("mode")
	if mode != "prod" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 'prod'"})
		return
	}

	// Find server config
	srvCfg := s.findServerConfig(req.Group, req.Server)
	if srvCfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	// 1. Determine etcd prefix
	configDirSuffix := filepath.Base(srvCfg.NginxConfigDir)
	etcdPrefix := path.Join(s.cfg.Sync.GitSyncer.KeyPrefix, req.Group, req.Server, configDirSuffix)

	// 2. Get pool and sync files to remote check directory
	pool, err := s.getPool(srvCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get SSH pool: %v", err)})
		return
	}

	remoteConfigDir := srvCfg.NginxConfigDir
	if remoteConfigDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("NginxConfigDir not find: %v", err)})
	}

	scpResult, err := ssh.ScpEtcdToRemote(c.Request.Context(), s.etcdClient, pool, srvCfg, etcdPrefix, remoteConfigDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to sync files to check directory: %v", err)})
		return
	}

	// 3. Run nginx test command using the config directory
	sshClient, err := pool.Get(srvCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get SSH client: %v", err)})
		return
	}
	defer pool.Put(sshClient)

	nginxBinary := srvCfg.NginxBinaryPath
	if nginxBinary == "" {
		nginxBinary = "nginx"
	}

	// Construct nginx -t command with custom config file
	// Assuming nginx.conf is at the root of the synced directory
	mainConfigFile := path.Join(remoteConfigDir, "nginx.conf")
	testCmd := fmt.Sprintf("%s -t -c %s", nginxBinary, mainConfigFile)

	output, err := sshClient.RunCommand(testCmd)
	success := err == nil

	c.JSON(http.StatusOK, UpdatePrepareResponse{
		Success: success,
		// Message: fmt.Sprintf("Found %d changes", len(changes)),
		// Changes: changes,
		Nginx: &NginxExecOutput{
			Command: testCmd,
			OK:      success,
			Output:  output,
		},
		Sync: &SyncResult{
			Total:        scpResult.Total,
			Skipped:      scpResult.Skipped,
			Added:        scpResult.Added,
			Updated:      scpResult.Updated,
			Deleted:      scpResult.Deleted,
			AddedFiles:   scpResult.AddedFiles,
			UpdatedFiles: scpResult.UpdatedFiles,
			DeletedFiles: scpResult.DeletedFiles,
		},
	})
}

func (s *Server) handleUpdateApply(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find server config
	srvCfg := s.findServerConfig(req.Group, req.Server)
	if srvCfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	pool, err := s.getPool(srvCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get SSH pool: %v", err)})
		return
	}

	configDirSuffix := filepath.Base(srvCfg.NginxConfigDir)
	gitPrefix := path.Join(s.cfg.Sync.GitSyncer.KeyPrefix, req.Group, req.Server, configDirSuffix)

	// 1. Upload files from etcd (production state)
	scpResult, err := ssh.ScpEtcdToRemote(c.Request.Context(), s.etcdClient, pool, srvCfg, gitPrefix, srvCfg.NginxConfigDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to sync files to remote: %v", err)})
		return
	}

	// 2. Reload Nginx
	sshClient, err := pool.Get(srvCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get SSH client: %v", err)})
		return
	}
	defer pool.Put(sshClient)

	// reloadCmd := srvCfg.ReloadCmd
	var reloadCmd string
	if srvCfg.NginxBinaryPath == "" {
		reloadCmd = "nginx -s reload"
	} else {
		reloadCmd = fmt.Sprint(srvCfg.NginxBinaryPath, " -s", " reload")
	}
	output, err := sshClient.RunCommand(reloadCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UpdateApplyResponse{
			Success: false,
			Message: "Failed to reload Nginx",
			Nginx: &NginxExecOutput{
				Command: reloadCmd,
				OK:      false,
				Output:  output,
			},
		})
		return
	}

	c.JSON(http.StatusOK, UpdateApplyResponse{
		Success: true,
		Message: fmt.Sprintf("Config applied (total: %d, updated: %d, skipped: %d) and Nginx reloaded",
			scpResult.Total, scpResult.Updated, scpResult.Skipped),
		Nginx: &NginxExecOutput{
			Command: reloadCmd,
			OK:      true,
			Output:  output,
		},
	})
}
