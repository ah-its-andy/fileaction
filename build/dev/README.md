# FileAction Development Guide

## 开发环境 Docker 镜像

本项目提供了专门用于开发调试的 Docker 镜像，支持源代码挂载和热重载。

### 可用的开发配置

#### 1. 基础开发环境 (使用 go run)

**Alpine 版本：**
```bash
docker-compose -f docker-compose.dev.yml --profile dev-alpine up
```

**Debian 版本：**
```bash
docker-compose -f docker-compose.dev.yml --profile dev-debian up
```

#### 2. 热重载开发环境 (使用 Air)

**Alpine 版本：**
```bash
docker-compose -f docker-compose.dev.yml --profile dev-air-alpine up
```

**Debian 版本：**
```bash
docker-compose -f docker-compose.dev.yml --profile dev-air-debian up
```

### 特性

- ✅ 源代码挂载到容器，修改代码无需重新构建镜像
- ✅ 使用 `go run` 直接运行，快速迭代
- ✅ 支持 Air 热重载，保存代码自动重启应用
- ✅ Go modules 缓存，加快依赖下载速度
- ✅ 同时支持 Alpine 和 Debian 基础镜像

### 目录挂载

开发容器会挂载以下目录：

- `.:/app` - 整个源代码目录
- `./config:/app/config` - 配置文件目录
- `./data:/app/data` - 数据目录
- `go-mod-cache` - Go modules 缓存卷

### 热重载配置

热重载功能使用 [Air](https://github.com/cosmtrek/air) 实现，配置文件为 `.air.toml`。

Air 会监控以下文件变化：
- `.go` 文件
- `.yaml`/`.yml` 配置文件
- `.html`/`.tpl`/`.tmpl` 模板文件

排除目录：
- `assets`, `tmp`, `vendor`, `testdata`, `data`, `logs`, `frontend`

### 停止容器

```bash
# 停止并移除容器
docker-compose -f docker-compose.dev.yml --profile dev-alpine down

# 或按 Ctrl+C 停止
```

### 清理

```bash
# 移除所有开发容器和卷
docker-compose -f docker-compose.dev.yml down -v
```

### 注意事项

1. **首次启动**：首次启动时会下载依赖，可能需要较长时间
2. **端口冲突**：开发环境使用端口 3000，确保该端口未被占用
3. **性能**：在 macOS/Windows 上，文件系统挂载可能影响性能，推荐使用 Docker Desktop 的最新版本
4. **权限问题**：容器内生成的文件可能有权限问题，可以在 Dockerfile 中设置合适的用户

### 构建开发镜像（可选）

如需单独构建开发镜像：

```bash
# Alpine 版本
docker build -f build/dev/Dockerfile.alpine -t fileaction-dev:alpine .

# Debian 版本
docker build -f build/dev/Dockerfile.debian -t fileaction-dev:debian .
```

### 与生产环境的区别

| 特性 | 开发环境 | 生产环境 |
|------|---------|---------|
| 镜像大小 | 大 (~800MB) | 小 (~20MB) |
| 启动方式 | `go run` | 编译后的二进制 |
| 代码修改 | 实时生效 | 需要重新构建 |
| 调试工具 | 包含 | 不包含 |
| 性能 | 较慢 | 快 |
| 用途 | 本地开发调试 | 生产部署 |
