# GitOps Nginx Management System

A GitOps-based system for managing Nginx configurations across multiple environments. This tool bridges the gap between Git repositories and remote Nginx servers, providing real-time synchronization, validation, and safe deployment workflows.

## Key Features

### 🔄 Real-time Synchronization
- **GitOps Core**: Treats your Git repository as the single source of truth.
- **Remote Polling**: Automatically fetches updates from remote Git repositories (Gitea/GitLab/GitHub).
- **Multi-Environment**: Supports separate "Production" and "Preview" environments.
- **Smart Rebase**: In "Rebase" mode, local changes are preserved and floated on top of remote updates, enabling safe hot-fixes.

### 🛡️ Safety & Validation
- **Dry-Run Checks**: Automatically runs `nginx -t` on remote servers before applying any changes.
- **Diff Preview**: Visualizes configuration differences between Git, Etcd, and Remote Servers (3-way diff).
- **Atomic Updates**: Ensures configuration files are synced and reloaded atomically.

### 🖥️ Modern Web UI
- **File Explorer**: Browse Nginx configuration files with status indicators (Modified/Added/Deleted).
- **Live Status**: Real-time view of Git synchronization status (Ahead/Behind/Diverged).
- **Visual Diff**: Built-in diff viewer to inspect changes line-by-line.
- **Operation Logs**: Track deployment history and results.

### ⚙️ Architecture
- **Backend**: Go (Golang) service utilizing `etcd` for state coordination and `go-git` for repository management.
- **Frontend**: React + Ant Design + Vite for a responsive management console.
- **Agentless**: Connects to Nginx servers via standard SSH/SFTP, requiring no agent installation on target nodes.

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 18+
- Etcd 3.5+
- Nginx servers with SSH access

### Configuration
1. Copy the example config:
   ```bash
   cp configs/config.yaml.example configs/config.yaml
   ```
2. Edit `configs/config.yaml` to define your Git repo, Nginx servers, and SSH credentials.

### Running
1. Start the backend:
   ```bash
   go run cmd/gitops-nginx/main.go
   ```
2. Start the frontend:
   ```bash
   cd ui
   npm install
   npm run dev
   ```

## Project Structure
For a detailed breakdown of the codebase, please refer to [PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md).

---
*Built with ❤️ for DevOps engineers.*