package ssh

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"maps"
	"path"
	"strings"
	"sync"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/pkg/log"
)

// ScpResult holds the summary of the sync operation.
type ScpResult struct {
	Total        int
	Skipped      int
	Updated      int
	Added        int
	Deleted      int
	UpdatedFiles []string
	AddedFiles   []string
	DeletedFiles []string
}

// ScpEtcdToRemote recursively and concurrently copies files from etcd prefix to remote server.
// It ensures strong consistency: deletes extra remote files, copies missing files, overwrites changed files.
func ScpEtcdToRemote(ctx context.Context, etcdCli *etcd.Client, pool *SFTPPool, srvCfg *config.ServerConfig, etcdPrefix string, remoteBaseDir string) (ScpResult, error) {
	var result ScpResult

	// 1. Get all files from etcd
	resp, err := etcdCli.GetPrefix(ctx, etcdPrefix)
	if err != nil {
		return result, fmt.Errorf("failed to get files from etcd: %w", err)
	}

	// Build map of etcd files: relPath -> content
	etcdFiles := make(map[string][]byte)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		// Skip metadata keys (.hash, .commit, .meta)
		if strings.HasSuffix(key, ".hash") || strings.HasSuffix(key, ".commit") || strings.HasSuffix(key, ".meta") {
			continue
		}
		relPath := strings.TrimPrefix(key, etcdPrefix)
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			continue
		}
		etcdFiles[relPath] = kv.Value
	}

	result.Total = len(etcdFiles)

	// 2. Get remote file list
	client, err := pool.Get(srvCfg)
	if err != nil {
		return result, fmt.Errorf("failed to get SFTP client from pool: %w", err)
	}

	remoteFiles, err := listRemoteFilesRecursive(client, remoteBaseDir, remoteBaseDir)
	pool.Put(client)
	if err != nil {
		return result, fmt.Errorf("failed to list remote files: %w", err)
	}

	// 3. Find files to delete (exist on remote but not in etcd)
	var filesToDelete []string
	for remoteRelPath := range remoteFiles {
		if _, exists := etcdFiles[remoteRelPath]; !exists {
			filesToDelete = append(filesToDelete, remoteRelPath)
		}
	}

	// 4. Delete extra remote files
	if len(filesToDelete) > 0 {
		delClient, err := pool.Get(srvCfg)
		if err != nil {
			return result, fmt.Errorf("failed to get SFTP client for deletion: %w", err)
		}
		for _, relPath := range filesToDelete {
			remotePath := path.Join(remoteBaseDir, relPath)
			if err := delClient.sftpClient.Remove(remotePath); err != nil {
				pool.Put(delClient)
				return result, fmt.Errorf("failed to delete remote file %s: %w", remotePath, err)
			}
			result.Deleted++
			result.DeletedFiles = append(result.DeletedFiles, relPath)
			log.Logger.WithField("prefix", etcdPrefix).WithField("remotePath", remotePath).Info("Deleted remote file")
		}
		pool.Put(delClient)
	}

	// 5. Sync files from etcd to remote (add or update)
	var wg sync.WaitGroup
	errChan := make(chan error, len(etcdFiles))
	sem := make(chan struct{}, 10) // Max 10 concurrent transfers
	var mu sync.Mutex

	for relPath, content := range etcdFiles {
		targetPath := path.Join(remoteBaseDir, relPath)

		wg.Add(1)
		go func(rPath, relP string, data []byte) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			// Calculate local hash
			hash := md5.Sum(data)
			localHash := hex.EncodeToString(hash[:])

			syncClient, err := pool.Get(srvCfg)
			if err != nil {
				errChan <- fmt.Errorf("failed to get SFTP client from pool: %w", err)
				return
			}
			defer pool.Put(syncClient)

			// Check if file exists on remote and compare hash
			remoteHash, exists := remoteFiles[relP]
			if exists && remoteHash == localHash {
				mu.Lock()
				result.Skipped++
				mu.Unlock()
				return
			}

			// Ensure remote directory exists
			dir := path.Dir(rPath)
			if err := syncClient.sftpClient.MkdirAll(dir); err != nil {
				errChan <- fmt.Errorf("failed to create remote directory %s: %w", dir, err)
				return
			}

			// Write file
			if err := syncClient.WriteFile(rPath, data); err != nil {
				errChan <- fmt.Errorf("failed to write remote file %s: %w", rPath, err)
				return
			}

			mu.Lock()
			if exists {
				result.Updated++
				result.UpdatedFiles = append(result.UpdatedFiles, relP)
			} else {
				result.Added++
				result.AddedFiles = append(result.AddedFiles, relP)
			}
			mu.Unlock()
		}(targetPath, relPath, content)
	}

	wg.Wait()
	close(errChan)

	// Collect first error if any
	for err := range errChan {
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// listRemoteFilesRecursive recursively lists all files under baseDir and returns a map of relPath -> md5hash.
func listRemoteFilesRecursive(client *Client, baseDir, currentDir string) (map[string]string, error) {
	files := make(map[string]string)

	entries, err := client.sftpClient.ReadDir(currentDir)
	if err != nil {
		// Directory doesn't exist, return empty map
		if strings.Contains(err.Error(), "not exist") || strings.Contains(err.Error(), "no such file") {
			return files, nil
		}
		return nil, err
	}

	// Recursively list files and directories
	for _, entry := range entries {
		fullPath := path.Join(currentDir, entry.Name())
		if entry.IsDir() {
			subFiles, err := listRemoteFilesRecursive(client, baseDir, fullPath)
			if err != nil {
				return nil, err
			}
			maps.Copy(files, subFiles)
		} else {
			relPath := strings.TrimPrefix(fullPath, baseDir)
			relPath = strings.TrimPrefix(relPath, "/")
			// Get file hash
			hash, err := client.GetFileHash(fullPath)
			if err != nil {
				hash = "" // If we can't get hash, treat as different
			}
			files[relPath] = hash
		}
	}

	return files, nil
}
