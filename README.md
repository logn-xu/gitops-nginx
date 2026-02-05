# GitOps Nginx Management System

[English](./README.md) | [简体中文](./README_CN.md)

A GitOps-based Nginx configuration management platform designed to bridge the gap between Git repositories and remote Nginx servers. It ensures configuration consistency, safety, and visibility across multiple environments (Production/Preview) through features like real-time synchronization, pre-release checks, and visual diffing. This project makes extensive use of vibe coding. If you find it helpful, please consider starring or submitting an issue.

## Key Features

### 🔄 Real-time Sync & GitOps
- **Single Source of Truth**: Treats your Git repository as the core, ensuring all configuration changes are traceable.
- **Remote Polling**: Automatically fetches the latest code from remote Git repositories (e.g., Gitea/GitHub).
- **Dual Sync Modes**:
  - **Reset Mode**: Forces consistency with the remote, ideal for strict production environments.
  - **Rebase Mode**: Prioritizes local modifications, suitable for development previews or quick production hotfixes.

### 🛡️ Safety & Validation
- **Pre-release Checks**: Automatically runs `nginx -t` syntax checks on remote servers before applying changes.
- **3-Way Diffing**: Visualizes differences between the Git repository, Etcd cache, and actual files on remote servers.
- **Atomic Release**: Ensures configuration file delivery and Nginx reload operations are executed atomically.

### 🖥️ Modern Web Console
- **Config Browser**: Tree view for configuration files with status indicators (Added/Modified/Deleted).
- **Git Status Dashboard**: Real-time view of sync status between local and remote branches (Behind/Ahead/Conflict).
- **Instant Diff**: Built-in code diff viewer with syntax highlighting.
- **Auto Refresh**: Supports configurable auto-refresh policies to stay aware of environment changes in real-time.

### ⚙️ Architecture
- **Backend**: Written in Go (Golang), using `etcd` for distributed coordination and `go-git` for version control.
- **Frontend**: Built with React + Ant Design + Vite.
- **Agentless**: Connects to Nginx servers via standard SSH/SFTP protocols, requiring no agent installation on target machines.

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 18+
- Etcd 3.5+
- Nginx servers with SSH access enabled

### Configuration
1. Copy the configuration template:
   ```bash
   cp configs/config.yaml.example configs/config.yaml
   ```
2. Modify `configs/config.yaml` to configure your Git repository, Nginx server list, and SSH authentication details.

### Starting the Service
1. Start the backend service:
   ```bash
   go run cmd/gitops-nginx/main.go apiserver
   ```
2. Start the frontend interface:
   ```bash
   cd ui
   npm install
   npm run dev
   ```

## Production Deployment
Refer to the production deployment documentation: [Production Deployment](./docs/DEPLOYMENT.md)

## Operating Guide
Refer to the operating guide documentation: [Operating Guide](./docs/OPERATING_GUIDE.md)


---
*An efficient configuration management tool built for DevOps engineers.*