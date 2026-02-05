# GitOps Nginx 管理系统 (GitOps Nginx Management System)

[English](./README.md) | [简体中文](./README_CN.md)

一个基于 GitOps 理念的 Nginx 配置管理平台，旨在连接 Git 仓库与远程 Nginx 服务器。它通过实时同步、预发布检查和可视化差异对比等功能，确保跨多个环境（生产/预览）的配置一致性、安全性和可见性。该项目大量使用了vibe coding代码。如果对你有帮助请发起issue或点个star。

## 核心功能

### 🔄 实时同步与 GitOps
- **单一事实来源**: 以 Git 仓库为核心，所有配置变更均可追溯。
- **远程轮询**: 自动从远程 Git 仓库（如 Gitea/GitHub）拉取最新代码。
- **双向模式**:
  - **Reset 模式**: 强制与远程保持一致，适合严格的生产环境。
  - **Rebase 模式**: 允许本地修改优先，适合开发预览或生产环境快速热修复 (Hotfix)。

### 🛡️ 安全与校验
- **预发布检查**: 在应用变更前，自动在远程服务器执行 `nginx -t` 语法检查。
- **三方差异对比**: 可视化对比 Git 仓库、Etcd 缓存与远程服务器实际文件之间的差异。
- **原子发布**: 确保配置文件的推送与 Nginx Reload 操作原子性执行。

### 🖥️ 现代化 Web 控制台
- **配置浏览器**: 树状视图展示配置文件，支持状态标记（新增/修改/删除）。
- **Git 状态看板**: 实时查看本地与远程分支的同步状态（落后/领先/冲突）。
- **即时 Diff**: 内置代码差异查看器，支持语法高亮。
- **自动刷新**: 界面支持可配置的自动刷新策略，实时感知环境变化。

### ⚙️ 技术架构
- **后端**: Go (Golang) 编写，使用 `etcd` 进行分布式协调，`go-git` 管理版本控制。
- **前端**: 基于 React + Ant Design + Vite 构建。
- **无 Agent**: 通过标准 SSH/SFTP 协议连接 Nginx 服务器，无需在目标机器安装任何代理软件。

## 快速开始

### 环境要求
- Go 1.22+
- Node.js 18+
- Etcd 3.5+
- 开启 SSH 访问权限的 Nginx 服务器

### 配置说明
1. 复制配置文件模板：
   ```bash
   cp configs/config.yaml.example configs/config.yaml
   ```
2. 修改 `configs/config.yaml`，配置您的 Git 仓库地址、Nginx 服务器列表及 SSH 认证信息。

### 启动服务
1. 启动后端服务：
   ```bash
   go run cmd/gitops-nginx/main.go apiserver
   ```
2. 启动前端界面：
   ```bash
   cd ui
   npm install
   npm run dev
   ```

## 生产部署
生产部署参考文档 [生产部署](./docs/DEPLOYMENT_CN.md)。

## 项目结构
关于详细的代码目录结构和模块说明，请参阅 [docs/PROJECT_STRUCTURE_CN.md](./docs/PROJECT_STRUCTURE_CN.md)。

---
*为运维工程师打造的高效配置管理工具。*
