package api

import "time"

// GroupSummary matches the frontend expectations
type GroupSummary struct {
	Name  string        `json:"name"`
	Hosts []HostSummary `json:"hosts"`
}

type HostSummary struct {
	Name            string `json:"name"`
	Host            string `json:"host"`
	ConfigDirSuffix string `json:"config_dir_suffix"`
}

type GroupsResponse struct {
	Groups []GroupSummary `json:"groups"`
}

type TreeResponse struct {
	Prefix       string            `json:"prefix"`
	Paths        []string          `json:"paths"`
	DiffPaths    []string          `json:"diff_paths,omitempty"`
	FileStatuses map[string]string `json:"file_statuses,omitempty"`
}

type TripleDiffResponse struct {
	Path           string `json:"path"`
	RemoteContent  string `json:"remote_content"`
	CompareContent string `json:"compare_content"`
	Diff           string `json:"diff"`
	Mode           string `json:"mode"`
	CompareLabel   string `json:"compare_label"`
	FileStatus     string `json:"file_status,omitempty"`
}

type CheckRequest struct {
	Server string `json:"server"`
	Group  string `json:"group"`
}

type CheckResponse struct {
	OK    bool             `json:"ok"`
	Mode  string           `json:"mode"`
	Sync  *SyncResult      `json:"sync,omitempty"`
	Nginx *NginxExecOutput `json:"nginx,omitempty"`
}

type UpdateRequest struct {
	Server string `json:"server"`
	Group  string `json:"group"`
}

type UpdatePrepareResponse struct {
	Success bool             `json:"success"`
	Nginx   *NginxExecOutput `json:"nginx,omitempty"`
	Sync    *SyncResult      `json:"sync,omitempty"`
	// Message string           `json:"message"`
	// Changes []string         `json:"changes,omitempty"`
}

type UpdateApplyResponse struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Nginx   *NginxExecOutput `json:"nginx,omitempty"`
}

type SyncResult struct {
	Total        int      `json:"total"`
	Skipped      int      `json:"skipped"`
	Added        int      `json:"added"`
	Updated      int      `json:"updated"`
	Deleted      int      `json:"deleted"`
	AddedFiles   []string `json:"added_files,omitempty"`
	UpdatedFiles []string `json:"updated_files,omitempty"`
	DeletedFiles []string `json:"deleted_files,omitempty"`
}

type NginxExecOutput struct {
	Command string `json:"command"`
	OK      bool   `json:"ok"`
	Output  string `json:"output"`
}

type GitStatusResponse struct {
	Branch       string      `json:"branch"`
	SyncMode     string      `json:"sync_mode"`
	LocalCommit  *CommitInfo `json:"local_commit,omitempty"`
	RemoteCommit *CommitInfo `json:"remote_commit,omitempty"`
	Status       string      `json:"status"` // "synced", "ahead", "behind", "diverged", "error"
	Diff         string      `json:"diff,omitempty"`
	Error        string      `json:"error,omitempty"`
}

type CommitInfo struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}
