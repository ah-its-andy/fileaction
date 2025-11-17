# 🚀 快速开始指南

## 配置 Docker Hub 自动构建

### 1️⃣ 获取 Docker Hub Token

```bash
# 访问 https://hub.docker.com/settings/security
# 创建新的 Access Token，权限选择 "Read, Write, Delete"
# 复制生成的 Token
```

### 2️⃣ 添加 GitHub Secrets

在 GitHub 仓库中添加以下 Secrets：

```
Settings → Secrets and variables → Actions → New repository secret
```

| Secret Name | Value |
|------------|-------|
| `DOCKERHUB_USERNAME` | 你的 Docker Hub 用户名 |
| `DOCKERHUB_TOKEN` | 刚才创建的 Access Token |

### 3️⃣ 触发构建

**方式一：推送代码到 main 分支**
```bash
git add .
git commit -m "Enable Docker auto-build"
git push origin main
```

**方式二：创建 Release**
```bash
# 在 GitHub 网页创建 Release
# 或使用命令行
git tag v1.0.0
git push origin v1.0.0
```

**方式三：手动触发**
```
GitHub Actions → Build and Push Docker Image → Run workflow
```

## 📦 构建的镜像标签

| Git 操作 | 生成的 Docker 标签 |
|---------|------------------|
| Push to main | `latest`, `main` |
| Tag `v1.0.0` | `1.0.0`, `1.0`, `1`, `latest` |
| PR | 仅构建不推送 |

## 🐳 使用构建的镜像

更新 `docker-compose.yml` 中的镜像名：

```yaml
services:
  fileaction:
    image: <你的用户名>/fileaction:latest  # 修改这里
    # ... 其他配置保持不变
```

然后运行：

```bash
docker-compose pull
docker-compose up -d
```

## ✅ 验证

1. 查看 GitHub Actions 运行状态
2. 检查 Docker Hub 仓库是否有新镜像
3. 本地拉取测试：

```bash
docker pull <你的用户名>/fileaction:latest
docker images | grep fileaction
```

## 📝 详细文档

查看完整配置说明：[DOCKER_SETUP.md](./DOCKER_SETUP.md)
