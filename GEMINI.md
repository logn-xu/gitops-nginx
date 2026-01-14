# GitOps Nginx Project Context

## Project Overview
This project is a **GitOps-based Nginx Management System** designed to bridge the gap between Git repositories and remote Nginx servers. It ensures configuration consistency, safety, and visibility across multiple environments (Production/Preview).

### Core Philosophy
- **Git as Source of Truth**: All configurations are managed via Git.
- **Agentless**: Operates via SSH/SFTP without agents on Nginx nodes.
- **Safety**: Implements dry-run checks (`nginx -t`) and atomic updates to prevent bad configurations from reaching production.

## Architecture
- **Backend**: Go (Golang) service.
  - **State**: Uses `etcd` for state coordination.
  - **Git**: Uses `go-git` for internal repository management.
- **Frontend**: React Single Page Application (SPA).
  - **Build Tool**: Vite.
  - **UI Framework**: Ant Design.
- **Connectivity**: Connects to Nginx servers via standard SSH/SFTP.

## Key Features
- **Real-time Synchronization**: Automatically fetches updates from remote Git repositories (Gitea/GitLab/GitHub).
- **Multi-Environment Support**: distinct handling for "Production" vs "Preview" environments.
- **Smart Rebase**: Preserves local hot-fixes by rebasing them on top of remote updates.
- **Visual Diff**: 3-way comparison between Git source, Etcd state, and actual Remote Server config.

## Development Setup

### Prerequisites
- **Go**: 1.22+
- **Node.js**: 18+
- **Database**: Etcd 3.5+
- **Infrastructure**: Nginx servers accessible via SSH

### Configuration
Configuration is managed in `configs/config.yaml`.
```bash
cp configs/config.yaml.example configs/config.yaml
# Edit config.yaml to define Git repo and Nginx server credentials
```

### Build & Run Commands

**Backend Service:**
```bash
go run cmd/gitops-nginx/main.go
```

**Frontend Application:**
```bash
cd ui
npm install
npm run dev
```

## Directory Status
*Note: As of the last analysis, this directory primarily contains documentation. The source code directories (`cmd/`, `ui/`, `configs/`) referenced in the documentation were not present in the current view. This context is derived from the project's README.*
