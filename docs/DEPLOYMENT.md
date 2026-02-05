# GitOps Nginx Production Deployment Guide

This document describes how to deploy the GitOps Nginx management system in a production environment.

## 1. Environment Preparation

### 1.1 Dependencies
- **Go**: 1.22+
- **Node.js**: 18+ (Only for manual frontend compilation; not required if using the embedded version)
- **Etcd**: 3.5+ (Cluster mode recommended for production)
- **Operating System**: Linux (e.g., Ubuntu, CentOS, Debian)

### 1.2 Nginx Server Requirements
- SSH access enabled.
- Recommended to configure a dedicated user (e.g., `nginx`) with write permissions for the Nginx configuration directory and execution permissions for `nginx -t` and `nginx -s reload`.

## 2. Compilation and Packaging

Use the `Makefile` in the project root for automated compilation.

### 2.1 One-Click Compilation (Backend + Embedded Frontend)
This command compiles the frontend and embeds the assets into the backend binary, producing a single-file executable.
```bash
make build-embed
```
The binary will be located at `bin/gitops-nginx`.

### 2.2 Multi-Platform Release Packaging
To generate release packages for various platforms:
```bash
make release-all
```
Packaged assets are found in `bin/release/`, containing `.tar.gz` archives for each platform.

### 2.3 Cleanup
```bash
make clean
```

## 3. Configuration

Configuration files are located in the `configs/` directory by default.

### 3.1 Main Configuration (config.yaml)
Copy the template and modify:
```bash
cp configs/config.example.yaml config.yaml
```
Key configuration items:
- `api.listen`: Service listening address.
- `etcd.endpoints`: Etcd cluster endpoints.
- `git`: Git repository URL, branch, sync mode (rebase/reset), and authentication.

### 3.2 Server Configuration (servers.yaml)
Define managed Nginx server groups:
```bash
cp configs/servers.example.yaml servers.yaml
```
Key configuration items:
- `nginx_servers`: Define server groups, host IPs, SSH authentication, Nginx paths, etc.

## 4. Production Execution

### 4.1 Recommended Directory Structure
Suggested organization on the production server:
```text
/app/gitops-nginx/
├── gitops-nginx (Executable binary)
├── config.yaml
├── servers.yaml
├── data/ (Used for Git repository clones)
└── logs/ (Log directory)
```

### 4.2 Start Command
```bash
./gitops-nginx ui
```
This starts both the API service and the embedded Web UI.

### 4.3 Using Systemd
Create `/etc/systemd/system/gitops-nginx.service`:
```ini
[Unit]
Description=GitOps Nginx Service
After=network.target remote-fs.target nss-lookup.target

[Service]
Type=simple
WorkingDirectory=/app/gitops-nginx/
ExecStart=/app/gitops-nginx/gitops-nginx ui
Restart=on-failure
RestartSec=5s
SyslogIdentifier=gitops-nginx
User=app
Group=app
Environment=GIN_MODE=release

# Security and performance optimization limits
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```
Start and enable at boot:
```bash
systemctl daemon-reload
systemctl enable gitops-nginx
systemctl start gitops-nginx
```

## 5. Configuration Management & Git Directory Structure

`gitops-nginx` tracks and manages Nginx servers based on the configuration in `servers.yaml`. To ensure correct synchronization, the Git repository directory structure must follow a specific hierarchy. If you need to collect configuration files from existing Nginx servers, you can use the companion tool [`ngx-collect`](https://github.com/logn-xu/ngx-collect).

### 5.1 Directory Mapping Rules

The directory mapping logic in the Git repository is as follows:
`{group}/{host}/conf/`

- **`group`**: Corresponds to the `group` field in `servers.yaml` (Server Group Name).
- **`host`**: Corresponds to the `host` field under the `servers` list in `servers.yaml` (Server IP).
- **`conf/`**: **Fixed directory**, stores all Nginx configuration files for that server. All content in this directory will be synced to the remote server's `nginx_config_dir`.

### 5.2 Configuration Example

Assume `servers.yaml` is configured as follows:

```yaml
nginx_servers:
  - group: "web-cluster"       # Server group name
    servers:
      - name: "nginx-node-01"  # Unique server identifier
        host: "192.168.1.10"   # Server IP
        port: 22 
        user: "nginx" 
        auth:
          # Options: password, key
          method: "password"
          password: "your_password"
        nginx_config_dir: "/app/nginx/conf"  # Target directory on the remote server
        nginx_binary_path: "/usr/sbin/nginx"
        check_dir: "/tmp/nginx_check"
```

### 5.3 Recommended Git Repository Structure

For the above configuration, the directory structure in the Git repository should be organized as:

```text
.
└── web-cluster/                # Matches 'group' in servers.yaml
    └── 192.168.1.10/           # Matches 'host' in servers.yaml
        └── conf/               # Fixed directory for config files
            ├── nginx.conf      # Main configuration file
            ├── conf.d/         # Additional configuration directory
            │   └── default.conf
            └── mime.types
```

**Synchronization Logic:**
- The system recursively scans all content under `{group}/{host}/conf/` in the Git repository.
- During sync, all files and subdirectories in this directory are distributed exactly as they are to the remote host's `nginx_config_dir`.
- **Important**: Ensure the directory hierarchy in Git matches exactly what the remote Nginx expects; otherwise, configuration loading may fail.

### 5.4 Quick Onboarding Advice

For existing Nginx servers, follow these steps to onboard:
1. Create the corresponding directory hierarchy in the Git repository: `mkdir -p {group}/{host}/conf`.
2. Copy all configuration files from `/app/nginx/conf/` on the existing Nginx server into this `conf/` directory.
3. Commit and push the Git changes.
4. The server's configuration will be automatically managed and visible in the `gitops-nginx` UI.


## 6. Operations

### 6.1 Configuration Validation
```bash
./gitops-nginx check
```
*Todo: Add SSH connectivity check.*

### 6.2 Hot Reload Configuration
When you modify the server list in `servers.yaml`, you can reload the configuration without restarting the service:
```bash
./gitops-nginx reload
```
This command sends a signal to the running process to trigger a configuration reload.

### 6.3 Security Recommendations
- **SSH Authentication**: It is strongly recommended to use `key` authentication instead of `password` in production environments.
