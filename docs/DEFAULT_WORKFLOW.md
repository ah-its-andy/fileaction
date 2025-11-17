# 默认工作流实现说明

## 实现内容

在数据库初始化时自动创建一个默认的JPEG到HEIC转换工作流。

## 修改的文件

### 1. backend/database/db.go

**添加的功能：**
- `initDefaultWorkflows()` 方法：在数据库初始化后自动创建默认工作流
- 检查数据库是否已有工作流，避免重复创建
- 嵌入完整的jpg-to-heic.yaml内容

**默认工作流详情：**
```yaml
name: convert-jpeg-to-heic
description: Convert JPEG images to HEIC format using ImageMagick
on:
  paths:
    - ./images
convert:
  from: jpeg
  to: heic
steps:
  - name: imagemagick-convert
    run: magick convert "${{ input_path }}" -quality 85 "${{ output_path }}"
    env:
      MAGICK_THREAD_LIMIT: "1"
  - name: verify-conversion
    run: file "${{ output_path }}" | grep -q "HEIC"
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.jpg"
  skip_on_nochange: true
```

### 2. Dockerfile

**添加的依赖：**
- `libwebp-tools` - WebP转换工具（cwebp命令）
- `pandoc` - 文档转换工具
- `texlive` - LaTeX引擎（PDF生成）
- `wget` - 健康检查

**添加的环境变量：**
- `PATH="/usr/bin:${PATH}"` - 确保命令可用
- `MAGICK_HOME="/usr"` - ImageMagick主目录
- `LD_LIBRARY_PATH="/usr/lib:${LD_LIBRARY_PATH}"` - 共享库路径

**命令验证：**
在构建时验证以下命令是否可用：
- `magick` (ImageMagick 7.x)
- `convert` (ImageMagick传统命令)
- `cwebp` (WebP编码器)
- `pandoc` (文档转换器)

### 3. 文档更新

**README.md:**
- 在Features部分添加"Default Workflow"特性
- 在Quick Start后添加默认工作流使用说明
- 说明如何快速开始使用预配置的工作流

**docs/QUICKSTART.md:**
- 在"First Workflow"之前添加"Using the Default Workflow"章节
- 提供分步指导使用默认工作流
- 包含创建测试图像的命令示例

**docs/PROJECT_SUMMARY.md:**
- 在数据库层描述中添加默认工作流初始化说明

**CHECKLIST.md:**
- 更新示例工作流部分，标注默认工作流自动创建

## 使用流程

### 首次启动

1. 用户首次启动FileAction
2. 数据库初始化创建所有表
3. `initDefaultWorkflows()` 检查workflows表是否为空
4. 如果为空，插入默认的JPEG转HEIC工作流
5. 工作流ID: `"default-jpeg-to-heic"`

### 用户体验

1. 用户打开 http://localhost:8080
2. 在Workflows视图中看到已存在的"convert-jpeg-to-heic"工作流
3. 创建 `./images` 目录并添加JPEG文件
4. 点击工作流的"🔍 Scan"按钮
5. 在Tasks视图中监控转换任务

## 优势

1. **零配置开始** - 用户无需手动创建工作流即可开始使用
2. **最佳实践示例** - 默认工作流展示了正确的YAML语法
3. **立即可用** - 对于常见的JPEG转HEIC需求，开箱即用
4. **教学价值** - 用户可以查看和学习工作流配置
5. **非侵入性** - 只在数据库为空时创建，不覆盖现有配置

## 技术细节

### 数据库检查逻辑

```go
// Check if any workflows exist
var count int
err := db.conn.QueryRow("SELECT COUNT(*) FROM workflows").Scan(&count)
if err != nil {
    return err
}

// If workflows already exist, skip initialization
if count > 0 {
    return nil
}
```

### 固定ID设计

使用固定ID `"default-jpeg-to-heic"` 而不是UUID的原因：
- 便于识别和引用
- 避免每次重新创建时ID变化
- 简化测试和文档

## 未来增强

可以考虑：
- [ ] 添加更多默认工作流（PNG转WebP、Markdown转PDF等）
- [ ] 允许用户通过配置文件选择要创建的默认工作流
- [ ] 提供工作流模板市场
- [ ] 支持从URL导入工作流

## 测试建议

1. **首次启动测试**
   ```bash
   rm -rf data/
   ./fileaction
   # 验证默认工作流已创建
   ```

2. **重复启动测试**
   ```bash
   ./fileaction
   # 验证不会创建重复工作流
   ```

3. **功能测试**
   ```bash
   mkdir -p images
   cp test.jpg images/
   # 通过UI扫描并执行
   ```

## 验收标准

✅ 首次启动时自动创建默认工作流
✅ 默认工作流包含完整的YAML配置
✅ 工作流已启用（enabled=true）
✅ 重复启动不会创建重复记录
✅ 用户可以正常使用默认工作流
✅ 文档已更新说明默认工作流
✅ Docker镜像包含所有必要的命令行工具

## 总结

通过在数据库初始化时嵌入默认的JPEG到HEIC转换工作流，FileAction提供了更好的开箱即用体验。用户无需手动配置即可立即开始使用文件转换功能，同时默认工作流也作为一个实用的示例，帮助用户理解如何创建自己的工作流。
