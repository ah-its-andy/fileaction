# Docker Hub 自动构建配置指南

本项目使用 GitHub Actions 自动构建 Docker 镜像并推送到 Docker Hub。

## 前置要求

1. Docker Hub 账号
2. GitHub 仓库管理权限

## 配置步骤

### 1. 创建 Docker Hub Access Token

1. 登录 [Docker Hub](https://hub.docker.com/)
2. 点击右上角头像 → **Account Settings**
3. 选择 **Security** 标签
4. 点击 **New Access Token**
5. 输入 Token 描述（例如：`github-actions`）
6. 选择权限：**Read, Write, Delete**
7. 点击 **Generate**
8. **复制并保存 Token**（只显示一次）

### 2. 在 GitHub 添加 Secrets

1. 进入你的 GitHub 仓库
2. 点击 **Settings** → **Secrets and variables** → **Actions**
3. 点击 **New repository secret** 添加以下两个 secrets：

   **Secret 1:**
   - Name: `DOCKERHUB_USERNAME`
   - Value: 你的 Docker Hub 用户名

   **Secret 2:**
   - Name: `DOCKERHUB_TOKEN`
   - Value: 刚才创建的 Access Token

### 3. 触发构建

配置完成后，以下操作会自动触发 Docker 镜像构建：

- **推送到 main 分支**: 生成 `latest` 标签
- **创建 Git Tag**: 例如 `v1.0.0` 会生成对应版本标签
- **Pull Request**: 构建但不推送
- **手动触发**: 在 Actions 页面点击 "Run workflow"

## Docker 镜像标签说明

构建的镜像会自动打上多个标签：

- `latest` - main 分支的最新版本
- `main` - main 分支构建
- `v1.0.0` - Git tag 版本（语义化版本）
- `v1.0` - 主版本号.次版本号
- `v1` - 主版本号
- `main-sha-abc123` - 分支名-commit sha

## 使用镜像

构建完成后，可以通过以下命令拉取镜像：

```bash
# 拉取最新版本
docker pull <你的用户名>/fileaction:latest

# 拉取指定版本
docker pull <你的用户名>/fileaction:v1.0.0
```

## 多平台支持

GitHub Actions 配置支持构建以下平台：
- `linux/amd64` (x86_64)
- `linux/arm64` (ARM64/Apple Silicon)

## 检查构建状态

1. 进入 GitHub 仓库的 **Actions** 页面
2. 查看 "Build and Push Docker Image" 工作流
3. 点击具体的运行记录查看详细日志

## 故障排查

### 认证失败
- 确认 `DOCKERHUB_USERNAME` 和 `DOCKERHUB_TOKEN` 正确设置
- 确认 Token 有足够的权限
- Token 可能已过期，需要重新生成

### 构建失败
- 查看 Actions 日志中的错误信息
- 确认 Dockerfile 语法正确
- 确认所有依赖都能正常安装

### 推送失败
- 确认 Docker Hub 仓库存在或有创建权限
- 确认网络连接正常
- 确认没有超出 Docker Hub 的速率限制
