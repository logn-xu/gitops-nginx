# GitOps Nginx 生产部署指南

本文档介绍如何在生产环境中部署 GitOps Nginx 管理系统。

## 1. 环境准备

### 1.1 依赖组件
- **Go**: 1.22+
- **Node.js**: 18+ (仅用于手动编译前端，若使用嵌入式服务可不单独部署)
- **Etcd**: 3.5+ (生产建议使用集群模式)
- **操作系统**: Linux (如 Ubuntu, CentOS, Debian)

### 1.2 Nginx 服务器要求
- 开启 SSH 访问权限
- 建议配置专用用户（如 `nginx`）并授予 Nginx 配置目录的写权限及 `nginx -t`, `nginx -s reload` 的执行权限。

## 2. 编译与打包

推荐使用项目根目录下的 `Makefile` 进行自动化编译。

### 2.1 一键编译 (后端 + 嵌入前端)
该命令会自动编译前端并将产物嵌入后端二进制文件中，生成单文件可执行程序。
```bash
make build-embed
```
编译产物位于 `bin/gitops-nginx`。

### 2.2 多平台发布打包
如果需要为不同平台生成发布包：
```bash
make release-all
```
打包产物位于 `bin/release/` 目录下，包含各平台的 `.tar.gz` 压缩包。

### 2.3 清理编译产物
```bash
make clean
```

## 3. 配置说明

项目配置文件默认位于 `configs/` 目录下。

### 3.1 主配置 (config.yaml)
复制模板并修改：
```bash
cp configs/config.example.yaml config.yaml
```
关键配置项：
- `api.listen`: 服务监听地址
- `etcd.endpoints`: Etcd 集群地址
- `git`: Git 仓库地址、分支、同步模式（rebase/reset）及认证信息

### 3.2 服务器配置 (servers.yaml)
定义受管理的 Nginx 服务器组：
```bash
cp configs/servers.example.yaml servers.yaml
```
关键配置项：
- `nginx_servers`: 定义服务器组、主机 IP、SSH 认证方式、Nginx 路径等。

## 4. 生产运行

### 4.1 目录结构建议
建议在生产服务器上按如下结构组织：
```text
/app/gitops-nginx/
├── gitops-nginx (二进制文件)
├── config.yaml
├── servers.yaml
├── data/ (用于存放 Git 仓库克隆)
└── logs/ (日志目录)
```

### 4.2 启动命令
```bash
./gitops-nginx ui
```
该命令会同时启动 API 服务和嵌入的 Web UI。

### 4.3 使用 Systemd 管理
创建 `/etc/systemd/system/gitops-nginx.service`:
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

# 安全与性能优化限制
# 限制最大文件打开数（类似于 ulimit -n）
LimitNOFILE=65535
# 限制最大进程数
LimitNPROC=4096

[Install]
```
启动并设置开机自启：
```bash
systemctl daemon-reload
systemctl enable gitops-nginx
systemctl start gitops-nginx
```

## 5. 配置管理与 Git 目录结构

`gitops-nginx` 通过 `servers.yaml` 中的配置来追踪和管理 Nginx 服务器。为了确保配置能够正确同步，Git 仓库的目录结构必须遵循特定的层级规范。如果需要从现有 Nginx 服务器收集配置文件，可以使用我的另外一个配套工具 [`ngx-collect`](https://github.com/logn-xu/ngx-collect)。

### 5.1 目录映射规则

Git 仓库的目录映射逻辑如下：
`{group}/{host}/conf/`

- **`group`**: 对应 `servers.yaml` 中的 `group` 字段（服务器组名）。
- **`host`**: 对应 `servers.yaml` 中 `servers` 列表下的 `host` 字段（服务器 IP）。
- **`conf/`**: **固定目录**，存放该服务器的所有 Nginx 配置文件。此目录下的所有内容将被同步到远程服务器的 `nginx_config_dir`。

### 5.2 配置示例

假设 `servers.yaml` 配置如下：

```yaml
nginx_servers:
  - group: "web-cluster"       # 服务器组名
    servers:
      - name: "nginx-node-01"  # 服务器唯一名称
        host: "192.168.1.10"
        port: 22 
        user: "nginx" 
        auth:
          # Options: password, key
          method: "password"
          password: "your_password"
        nginx_config_dir: "/app/nginx/conf"  # 同步到远程的该目录
        nginx_binary_path: "/usr/sbin/nginx"
        check_dir: "/tmp/nginx_check"
```

### 5.3 Git 仓库推荐结构

对应上述配置，Git 仓库中的目录结构应组织如下：

```text
.
└── web-cluster/                # 对应 servers.yaml 中的 group
    └── 192.168.1.10/          # 对应 servers.yaml 中的 host
        └── conf/               # 固定存放配置的目录
            ├── nginx.conf      # 主配置文件
            ├── conf.d/         # 额外的配置目录
            │   └── default.conf
            └── mime.types
```

**同步逻辑说明：**
- 系统会自动将 Git 仓库中 `{group}/{host}/conf/` 目录下的所有文件和子目录递归同步到远程主机的 `nginx_config_dir`。
- **重要提示**：请确保 Git 中的文件层级与远程 Nginx 期望的层级完全一致。

### 5.4 快速初始化建议

对于现有的 Nginx 服务器，建议按以下步骤接入：
1. 在 Git 仓库中创建对应的目录层级：`mkdir -p {group}/{host}/conf`。
2. 将现有Nginx服务器上 `/app/nginx/conf/` 下的所有配置文件拷贝到该 `conf/` 目录下。
3. 提交并推送 Git 变更。
4. 在 `gitops-nginx` UI 界面中即可看到该服务器的配置已纳入管理。


## 6. 运维操作

### 6.1 配置校验
```bash
./gitops-nginx check
```
Todo: ssh连通性检查。

### 6.2 热重载配置
当你修改了 `servers.yaml` 中的服务器列表后，无需重启服务，可以执行：
```bash
./gitops-nginx reload
```
该命令会向运行中的进程发送信号，触发配置重载。

### 6.2 安全建议
- **SSH 认证**: 生产环境强烈建议使用 `key` 认证而非 `password`。
