package sync

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	"github.com/logn-xu/gitops-nginx/internal/etcd"
	"github.com/logn-xu/gitops-nginx/pkg/log"
	"github.com/sirupsen/logrus"
)

// IsIgnored checks if a file path matches any ignore pattern
// It automatically ignores .git directory and hidden files/directories starting with .
// unless they are explicitly not ignored (though we enforce hidden file ignore for now based on requirements)
// The requirements say: "filter out .swp . hidden files".
func IsIgnored(filePath string, ignorePatterns []string) bool {
	filename := filepath.Base(filePath)

	// Always ignore .git directory
	if strings.Contains(filePath, ".git/") || filename == ".git" {
		return true
	}

	// Always ignore hidden files (starting with .)
	// This covers .DS_Store, .gitignore, .env, etc.
	if strings.HasPrefix(filename, ".") {
		return true
	}

	// Always ignore swap files (ending with .swp)
	if strings.HasSuffix(filename, ".swp") {
		return true
	}

	for _, pattern := range ignorePatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
		// Also check if path contains the pattern (for directory patterns)
		if strings.Contains(filePath, pattern) {
			return true
		}
	}
	return false
}

// mirrorDeleteEtcdPrefix removes keys from etcd that have a certain prefix but are not in the provided map of relative paths.
// This ensures that etcd remains a mirror of the remote server's configuration by cleaning up deleted files.
func mirrorDeleteEtcdPrefix(ctx context.Context, etcdClient *etcd.Client, prefix string, relPaths map[string]struct{}) error {
	// Construct a map of all allowed keys (original file, hash, commit, meta)
	allowed := make(map[string]struct{}, len(relPaths)*4)
	for rel := range relPaths {
		base := path.Join(prefix, rel)
		allowed[base] = struct{}{}
		allowed[base+".hash"] = struct{}{}
		allowed[base+".commit"] = struct{}{}
		allowed[base+".meta"] = struct{}{}
	}

	// Retrieve all keys with the specified prefix from etcd
	resp, err := etcdClient.GetPrefix(ctx, prefix)
	if err != nil {
		return err
	}

	deleted := 0
	for _, kv := range resp.Kvs {
		k := string(kv.Key)
		// Skip the prefix itself if it's an exact match
		if k == prefix {
			continue
		}
		// Ensure the key strictly starts with the prefix (redundant but safe)
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		// If the key is not in the allowed map, it means it's an orphan and should be deleted
		if _, ok := allowed[k]; ok {
			continue
		}
		if _, err := etcdClient.Delete(ctx, k); err == nil {
			deleted++
		}
	}

	// Log the cleanup activity if any keys were deleted
	if deleted > 0 {
		log.Logger.WithFields(logrus.Fields{
			"prefix":  prefix,
			"deleted": deleted,
		}).Info("mirror deleted extra etcd keys")
	}

	return nil
}
