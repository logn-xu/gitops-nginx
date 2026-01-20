package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitrepo "github.com/logn-xu/gitops-nginx/internal/git"
)

func (s *Server) handleGetGitStatus(c *gin.Context) {
	repo, err := gitrepo.OpenRepository(&s.cfg.Git)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to open repo: %v", err)})
		return
	}

	branchName := s.cfg.Git.Branch
	if branchName == "" {
		branchName = "master"
	}

	// 1. Get Local Commit (HEAD)
	headRef, err := repo.Head()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get HEAD: %v", err)})
		return
	}
	localCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get local commit: %v", err)})
		return
	}

	// 2. Get Remote Commit (origin/branch)
	remoteRefName := plumbing.NewRemoteReferenceName("origin", branchName)
	remoteRef, err := repo.Reference(remoteRefName, true)

	var remoteCommit *object.Commit
	if err == nil {
		remoteCommit, err = repo.CommitObject(remoteRef.Hash())
		if err != nil {
			// Failed to get remote commit object, could be missing in local clone
			// We continue with remoteCommit as nil
		}
	}

	response := GitStatusResponse{
		Branch:      branchName,
		SyncMode:    "git",
		LocalCommit: toCommitInfo(localCommit),
		Status:      "unknown",
	}

	if remoteCommit != nil {
		response.RemoteCommit = toCommitInfo(remoteCommit)

		// 3. Compare Status
		if localCommit.Hash == remoteCommit.Hash {
			response.Status = "synced"
		} else {
			isLocalBehind := false
			isLocalAhead := false

			// Check if local is ancestor of remote (Behind)
			if ok, err := localCommit.IsAncestor(remoteCommit); err == nil && ok {
				isLocalBehind = true
			}

			// Check if remote is ancestor of local (Ahead)
			if ok, err := remoteCommit.IsAncestor(localCommit); err == nil && ok {
				isLocalAhead = true
			}

			if isLocalAhead && !isLocalBehind {
				response.Status = "ahead"
			} else if isLocalBehind && !isLocalAhead {
				response.Status = "behind"
			} else {
				response.Status = "diverged"
			}

			// 4. Generate Diff (Remote -> Local)
			patch, err := remoteCommit.Patch(localCommit)
			if err == nil {
				response.Diff = patch.String()
			} else {
				response.Error = fmt.Sprintf("failed to generate diff: %v", err)
			}
		}
	} else {
		response.Status = "error"
		response.Error = "Remote reference not found. Please ensure the repository is synced."
	}

	c.JSON(http.StatusOK, response)
}

func toCommitInfo(c *object.Commit) *CommitInfo {
	if c == nil {
		return nil
	}
	return &CommitInfo{
		Hash:      c.Hash.String(),
		Message:   c.Message,
		Author:    c.Author.Name,
		Timestamp: c.Author.When,
	}
}
