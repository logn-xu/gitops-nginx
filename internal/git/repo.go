package git

import (
	"fmt"

	"os"

	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/logn-xu/gitops-nginx/internal/config"
	"golang.org/x/crypto/ssh"
)

var syncMu sync.Mutex

// Repository is a git repository
type Repository struct {
	*git.Repository
}

// SyncRepository ensures the local repository is cloned and synced to the specified branch.
func SyncRepository(cfg *config.GitConfig) (*Repository, error) {
	syncMu.Lock()
	defer syncMu.Unlock()

	branch := cfg.Branch
	if branch == "" {
		branch = "master"
	}

	auth, err := getAuth(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get git auth: %w", err)
	}

	_, err = os.Stat(cfg.RepoPath)
	if os.IsNotExist(err) {
		// Repository does not exist, clone it
		r, err := git.PlainClone(cfg.RepoPath, false, &git.CloneOptions{
			URL:           cfg.RepoURL,
			Auth:          auth,
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			Progress:      os.Stdout,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository from %s to %s: %w", cfg.RepoURL, cfg.RepoPath, err)
		}
		return &Repository{r}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat repository path %s: %w", cfg.RepoPath, err)
	}

	// Repository exists, open it and pull changes
	r, err := git.PlainOpen(cfg.RepoPath)
	if err != nil {
		// If opening fails (e.g. not a git repo), try to remove and re-clone
		if err == git.ErrRepositoryNotExists {
			if err := os.RemoveAll(cfg.RepoPath); err != nil {
				return nil, fmt.Errorf("failed to remove non-git directory: %w", err)
			}
			return SyncRepository(cfg)
		}
		return nil, fmt.Errorf("failed to open repository at %s: %w", cfg.RepoPath, err)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Pull changes from remote
	err = w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		Force:         true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("failed to pull changes: %w", err)
	}

	return &Repository{r}, nil
}

// getAuth returns the authentication method for the git repository
func getAuth(cfg *config.GitConfig) (transport.AuthMethod, error) {
	switch cfg.Auth.Type {
	case "basic":
		return &http.BasicAuth{
			Username: cfg.Auth.Username,
			Password: cfg.Auth.Password,
		}, nil
	case "ssh":
		if cfg.Auth.PrivateKeyPath == "" {
			return nil, fmt.Errorf("SSH private key path is required")
		}
		publicKeys, err := gitssh.NewPublicKeysFromFile("git", cfg.Auth.PrivateKeyPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load SSH private key: %w", err)
		}
		// Ignore host key for simplicity, in production you should use a fixed host key
		publicKeys.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		return publicKeys, nil
	case "none", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported git auth type: %s", cfg.Auth.Type)
	}
}

// OpenRepository opens a git repository without syncing
func OpenRepository(config *config.GitConfig) (*Repository, error) {
	r, err := git.PlainOpen(config.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository at %s: %w", config.RepoPath, err)
	}
	return &Repository{r}, nil
}

// GetFileContent returns the content of a file at a specific commit.
func (r *Repository) GetFileContent(filePath string, commitHash plumbing.Hash) (string, error) {
	commit, err := r.CommitObject(commitHash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit object %s: %w", commitHash, err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get tree from commit %s: %w", commitHash, err)
	}

	file, err := tree.File(filePath)
	if err != nil {
		if err == object.ErrFileNotFound {
			return "", err // Return the specific error for the caller to handle
		}
		return "", fmt.Errorf("failed to get file '%s' from tree: %w", filePath, err)
	}

	content, err := file.Contents()
	if err != nil {
		return "", fmt.Errorf("failed to read file content for '%s': %w", filePath, err)
	}

	return content, nil
}

// GetHeadHash returns the hash of the HEAD commit.
// The original method was named Head() which caused a recursive call.
func (r *Repository) GetHeadHash() (plumbing.Hash, error) {
	ref, err := r.Repository.Head() // Correctly call the embedded repository's Head method
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to get HEAD reference: %w", err)
	}
	return ref.Hash(), nil
}
