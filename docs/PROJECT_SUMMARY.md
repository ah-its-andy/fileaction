# FileAction - 项目实现总结

## 项目概述

FileAction是一个轻量级工作流自动化引擎，灵感来源于GitHub Actions，专注于文件格式转换和批量处理。项目采用纯Go后端（无CGO依赖）和Fiber框架 + 原生JavaScript混合架构的前端。

## 技术栈

### 后端
- **Go 1.21+**: 主要编程语言
- **Fiber v2**: 高性能HTTP框架
- **modernc.org/sqlite**: 纯Go实现的SQLite（无CGO依赖）
- **gopkg.in/yaml.v3**: YAML解析
- **github.com/google/uuid**: UUID生成
- **github.com/robfig/cron/v3**: 定时任务（可选）

### 前端
- **Vanilla JavaScript**: 无框架，纯原生JS
- **HTML5 + CSS3**: 现代Web标准
- **Fetch API**: HTTP请求
- **Hash-based Routing**: 单页应用路由

### 部署
- **Docker**: 容器化部署
- **Docker Compose**: 编排配置
- **Make**: 构建自动化

## 核心功能实现

### 1. 数据库层 (backend/database/)

**实现内容:**
- 使用modernc.org/sqlite的纯Go SQLite实现
- 四个主要表：workflows, files, tasks, task_steps
- Repository模式封装CRUD操作
- 事务支持和索引优化
- WAL模式提升并发性能

**关键文件:**
- `db.go`: 数据库连接和初始化
- `workflow_repo.go`: 工作流仓库
- `file_repo.go`: 文件仓库
- `task_repo.go`: 任务仓库
- `task_step_repo.go`: 任务步骤仓库

**默认工作流初始化:**
- 首次启动时自动创建JPEG到HEIC转换工作流
- 只在数据库为空时创建，避免重复
- 使用UUID作为默认工作流ID

### 2. 工作流引擎 (backend/workflow/)

**实现内容:**
- YAML工作流定义解析
- 变量替换系统（${{ variable }}）
- 输出路径生成逻辑
- 文件glob模式匹配
- 工作流验证

**关键功能:**
- 支持多路径扫描
- 可配置的并发度
- 文件扩展名转换
- 环境变量支持

### 3. 文件扫描器 (backend/scanner/)

**实现内容:**
- 递归目录扫描
- MD5哈希计算
- 文件索引更新
- 智能任务创建（基于MD5变化）

**特性:**
- 支持子目录扫描开关
- 文件glob过滤
- skip_on_nochange模式
- 增量扫描支持

### 4. 任务执行引擎 (backend/executor/)

**实现内容:**
- 工作池模式
- Shell命令执行
- 日志管理（运行时文本文件，完成后导入DB）
- 超时控制
- 任务取消支持

**特性:**
- 可配置工作线程数
- 每个任务和步骤的独立超时
- stdout/stderr捕获
- 退出码记录
- 环境变量传递

### 5. HTTP API服务器 (backend/api/)

**实现内容:**
- 基于Fiber框架的RESTful API
- 工作流CRUD端点
- 任务管理端点
- 文件列表端点
- 日志尾部跟踪端点

**特性:**
- CORS支持
- 错误处理中间件
- 日志中间件
- 静态文件服务
- 分页支持

### 6. 前端SPA (frontend/)

**实现内容:**
- 纯JavaScript单页应用
- Hash路由系统
- GitHub Actions风格UI
- 实时日志查看器

**页面:**
- Workflows: 工作流管理（CRUD）
- Tasks: 任务列表（过滤、分页）
- Files: 文件浏览器

**特性:**
- 无框架依赖
- 实时日志轮询
- 响应式设计
- 暗色主题

## 项目结构

```
fileaction/
├── backend/
│   ├── api/              # HTTP服务器和API处理器
│   ├── config/           # 配置管理
│   ├── database/         # 数据库层
│   ├── executor/         # 任务执行引擎
│   ├── models/           # 数据模型
│   ├── scanner/          # 文件扫描器
│   └── workflow/         # 工作流解析器
├── frontend/
│   ├── index.html        # 主HTML
│   ├── style.css         # 样式
│   └── app.js            # JavaScript逻辑
├── config/
│   └── config.yaml       # 配置文件
├── docs/
│   ├── example-workflows/ # 示例工作流
│   ├── API.md            # API文档
│   ├── DEVELOPMENT.md    # 开发指南
│   └── QUICKSTART.md     # 快速开始
├── .github/
│   └── workflows/
│       └── ci.yml        # CI配置
├── main.go               # 入口文件
├── go.mod                # Go模块
├── Makefile              # 构建脚本
├── Dockerfile            # Docker镜像
├── docker-compose.yml    # Docker Compose
└── README.md             # 项目说明
```

## 关键设计决策

### 1. 纯Go实现（无CGO）
- 使用modernc.org/sqlite代替cgo sqlite3
- 确保跨平台编译
- 简化部署流程

### 2. Fiber框架
- 高性能HTTP服务器
- Express风格的API
- 丰富的中间件支持

### 3. 原生JavaScript前端
- 无构建步骤
- 零外部依赖
- 快速加载
- 易于理解和维护

### 4. 日志处理策略
- 运行时写入文本文件
- 完成后导入数据库
- 支持实时尾部跟踪
- 避免数据库频繁写入

### 5. MD5变化检测
- 高效的文件变化识别
- 避免重复处理
- 支持增量更新

## API端点

### 工作流
- `GET /api/workflows` - 列出所有工作流
- `POST /api/workflows` - 创建工作流
- `GET /api/workflows/:id` - 获取工作流详情
- `PUT /api/workflows/:id` - 更新工作流
- `DELETE /api/workflows/:id` - 删除工作流
- `POST /api/workflows/:id/scan` - 触发扫描

### 任务
- `GET /api/tasks` - 列出任务（支持过滤）
- `GET /api/tasks/:id` - 获取任务详情
- `GET /api/tasks/:id/steps` - 获取任务步骤
- `GET /api/tasks/:id/log/tail` - 尾部跟踪日志
- `POST /api/tasks/:id/retry` - 重试任务
- `POST /api/tasks/:id/cancel` - 取消任务
- `DELETE /api/tasks/:id` - 删除任务

### 文件
- `GET /api/files?workflow_id=:id` - 列出文件

## 示例工作流

### 1. JPEG转HEIC
```yaml
name: convert-jpeg-to-heic
on:
  paths:
    - ./images
convert:
  from: jpeg
  to: heic
steps:
  - name: imagemagick-convert
    run: magick convert "${{ input_path }}" -quality 85 "${{ output_path }}"
options:
  concurrency: 2
  file_glob: "*.jpg"
  skip_on_nochange: true
```

### 2. PNG转WebP
```yaml
name: convert-png-to-webp
on:
  paths:
    - ./images/png
convert:
  from: png
  to: webp
steps:
  - name: convert-to-webp
    run: cwebp -q 85 "${{ input_path }}" -o "${{ output_path }}"
options:
  concurrency: 4
  file_glob: "*.png"
```

### 3. Markdown转PDF
```yaml
name: markdown-to-pdf
on:
  paths:
    - ./documents
convert:
  from: md
  to: pdf
steps:
  - name: pandoc-convert
    run: pandoc "${{ input_path }}" -o "${{ output_path }}" --pdf-engine=xelatex
options:
  concurrency: 2
  file_glob: "*.md"
```

## 测试覆盖

### 单元测试
- `backend/workflow/parser_test.go`: 工作流解析测试
- `backend/database/db_test.go`: 数据库CRUD测试

### 集成测试
- 完整工作流执行
- API端点测试
- 文件扫描测试

## 构建和部署

### 本地构建
```bash
make build
./fileaction
```

### Docker部署
```bash
docker-compose up -d
```

### 跨平台编译
```bash
# Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o fileaction-linux .

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o fileaction.exe .
```

## 配置选项

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "./data/fileaction.db"

logging:
  dir: "./data/logs"

execution:
  default_concurrency: 4
  task_timeout: 3600s
  step_timeout: 1800s

polling:
  interval: 2s
```

## 性能特性

- **并发处理**: 可配置工作池
- **智能跳过**: MD5变化检测
- **高效数据库**: SQLite WAL模式
- **轻量前端**: 无框架快速加载
- **单一二进制**: 无外部依赖

## 安全考虑

- Shell命令执行风险（需要用户信任）
- 默认无认证（生产环境需要反向代理）
- 文件系统访问控制
- CORS配置

## 未来增强

- [ ] WebSocket实时更新
- [ ] 工作流模板市场
- [ ] 批量操作支持
- [ ] 认证和授权
- [ ] 监控和指标
- [ ] 插件系统
- [ ] 多节点支持

## 验收标准完成情况

✅ 解析并执行有效的YAML工作流
✅ 扫描目录，创建文件索引，基于MD5生成任务
✅ 并发执行任务，日志写入文本文件，完成后导入DB
✅ 通过轮询为运行任务提供实时日志查看
✅ 在SPA中完全支持工作流和任务的CRUD
✅ 支持跨工作流的"ALL"任务视图
✅ 处理失败、重试和取消
✅ 代码是纯Go（no cgo）和Fiber + 原生JS混合架构
✅ 提供示例工作流（jpg-to-heic.yaml等）
✅ Docker和docker-compose部署支持

## 总结

FileAction项目成功实现了所有核心需求，提供了一个完整、可用的工作流自动化引擎。项目采用现代化的技术栈，注重性能和可维护性，适合用于文件格式转换、批量处理等场景。

主要亮点：
- 纯Go实现，跨平台兼容
- 高性能Fiber框架
- 简洁的原生JS前端
- 完善的文档和示例
- Docker化部署支持
- 可扩展的架构设计

项目已经具备生产环境部署的基础，可以根据实际需求进行进一步的定制和优化。
