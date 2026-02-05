# GitOps-Nginx Operating Guide

## Project Introduction

`gitops-nginx` is an Nginx configuration management platform designed specifically for DevOps and developers. Based on the **GitOps** core philosophy, it uses a Git repository as the "Single Source of Truth" to achieve versioning, automated validation, and secure distribution of Nginx configurations.

This tool aims to solve the following problems caused by traditional manual modification of Nginx configurations:
- **Untraceable**: Difficult to audit modification records.
- **Validation Risks**: Service downtime caused by missing `nginx -t` before deployment.
- **Environment Consistency**: Difficulties in synchronizing configurations across multiple servers.

### Core Features

- **Real-time Perception**: Utilizes `fsnotify` to listen for local changes and provide real-time feedback to the Web interface.
- **Security Pre-check**: Automatically performs remote isolated pre-checks on the target machine before applying configurations.
- **Visual Difference**: Intuitively displays configuration differences between local, Git repository, and production environments.

---

## Quick Start: Two Configuration Change Modes

`gitops-nginx` supports flexible workflows to meet different scenarios from "quick debugging" to "strict deployment."

### 1. Rapid Debugging Mode: Based on Local File Modification

This mode is suitable for development or testing phases, where files are modified directly on the server where the console is located for immediate results.

#### Preview Phase (Preview Mode)
Preview mode allows you to debug configurations in real-time as if you were on a remote Nginx server.

![Preview Mode Interface](pic/pic1.png)

Server directory structure is as follows:

![Config File Directory](pic/pic2.png)

- **Operation**: You can directly use `vim` to modify files in the Git workspace directory (e.g., `/app/repo/`) on the server where `gitops-nginx` is located.
- **Real-time Sync**: Any save action will be immediately pushed to the frontend via `fsnotify`.

![Modify Config File](pic/pic3.png)

**File Status Indicators**:
- <span style="color: #faad14">**★ Modified**</span>: File content has changed.
- <span style="color: #52c41a">**+ Added**</span>: Newly created configuration file.
- <span style="color: #ff4d4f">**- Deleted**</span>: File has been removed.

![Change Status Display](pic/pic4.png)

#### Remote Pre-check (Pre-check)
In Preview mode, clicking the "Check" button will recursively synchronize all changes from the current workspace to a **temporary isolated directory** on the target Nginx server (e.g., `/app/nginx/.check/`) and execute `nginx -t`.

![Config Validation Principle](pic/pic5.png)

- **Security**: The validation process does not affect the existing running state of Nginx.
- **Feedback**: If there is a syntax error (e.g., a missing semicolon), the console will directly display the Nginx error details.

![Error Example](pic/pic6.png)

#### Production Application (Production Mode)
Once you have confirmed the configuration is correct in Preview mode, you need to `commit` the changes to the local repository before they can be applied in "Production Mode."

Local Commit:

![git commit](pic/pic7.png)

Ahead Status:

![Ahead Status](pic/pic8.png)

Click Update Pre-check:

![Update Pre-check](pic/pic9.png)

Apply Update:

![Apply Update Process](pic/pic10.png)

1. **Commit Changes**: Execute `git add . && git commit -m "update config"` in the server terminal.
2. **Status Perception**: The Web interface will detect the local commit and show the repository status as **Ahead**.
3. **Execute Update**: Click the "Update" button; the system will perform a final production pre-check and formally reload the remote Nginx service.

---

### 2. Standard GitOps Workflow: Based on Remote Git Submission

This is the most recommended production workflow, suitable for multi-person collaboration and auditing.

1. **Local Development**: Manage Nginx configurations via Git on your own computer.
2. **Local Commit & Push**: Push modifications to a remote Git repository (e.g., GitHub/GitLab/Gitea).
3. **Automatic Pull**: `gitops-nginx` will automatically pull remote changes based on the configured polling interval.
4. **One-click Go-live**: Confirm differences in Production Mode and click apply.

![VSCode Modification](pic/pic11.png)

![VSCode Submit Config](pic/pic12.png)

![Git Workflow Demo](pic/pic13.png)

![Config Update](pic/pic14.png)

Validation of update results is shown below:

![Config Update Result](pic/pic15.png)

---

## Roadmap

We are continuously optimizing; the following features will be available soon:
- [ ] **Version Persistence**: Record the specific Release version currently running in Nginx.
- [ ] **Release History Audit**: Detailed records of who operated which server and when.
- [ ] **Quick Rollback**: Support second-level rollback based on `git revert` or snapshots.
- [ ] **Multi-branch Canary**: Support applying configurations from different branches to different server groups.

---

## Best Practice Recommendations

- **Production Environment**: Be sure to use **SSH Key** for authentication to avoid storing passwords in plain text in configuration files.
- **Config Isolation**: It is recommended to divide different Nginx server groups for different business units and use folders for isolation in the Git repository.
- **Regular Backup**: Although the tool provides synchronization features, regular backups of `/etc/nginx` on the target machine remain a good habit.
