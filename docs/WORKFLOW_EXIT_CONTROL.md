# Workflow 退出控制机制

## 概述

FileAction 支持在 workflow 的 `run` 命令中使用特殊的退出码来控制任务的执行流程和最终状态。

## 退出码说明

| 退出码 | 含义 | 步骤状态 | 任务状态 | 后续步骤 |
|--------|------|---------|---------|---------|
| **0** | 步骤成功 | completed | 继续执行 | 继续下一步 |
| **100** | 成功并停止 workflow | completed | **completed** | 停止执行 |
| **101** | 失败并停止 workflow | failed | **failed** | 停止执行 |
| **其他非零值** | 步骤失败 | failed | failed | 停止执行 |

## 使用场景

### 1. 条件跳过（成功退出）

当检测到某些条件时，可以跳过后续处理并标记任务为成功：

```yaml
steps:
  - name: check-prerequisites
    run: |
      # 检查文件是否已经处理过
      if [ -f "${{ output_path }}" ]; then
        echo "File already processed, skipping"
        exit 100  # 成功退出，跳过后续步骤
      fi
      echo "File needs processing"

  - name: process-file
    run: |
      # 只有上一步返回 0 才会执行这里
      convert "${{ input_path }}" "${{ output_path }}"
```

### 2. 文件大小限制

```yaml
steps:
  - name: check-file-size
    run: |
      file_size=$(stat -c%s "${{ input_path }}")
      
      # 文件为空，跳过处理（成功）
      if [ "$file_size" -eq 0 ]; then
        echo "Empty file, skipping"
        exit 100
      fi
      
      # 文件太大，无法处理（失败）
      if [ "$file_size" -gt 104857600 ]; then
        echo "File too large (>100MB)"
        exit 101
      fi

  - name: process
    run: |
      # 处理文件
      process_command "${{ input_path }}" "${{ output_path }}"
```

### 3. 内容验证

```yaml
steps:
  - name: validate-format
    run: |
      # 检查文件格式
      file_type=$(file -b "${{ input_path }}")
      
      if echo "$file_type" | grep -qi "jpeg"; then
        echo "Valid JPEG file"
        exit 0  # 继续处理
      elif echo "$file_type" | grep -qi "already converted"; then
        echo "File already in target format"
        exit 100  # 成功跳过
      else
        echo "Invalid file format: $file_type"
        exit 101  # 格式错误，标记失败
      fi

  - name: convert
    run: magick "${{ input_path }}" -quality 85 "${{ output_path }}"
```

### 4. 智能重试控制

```yaml
steps:
  - name: check-retry-count
    run: |
      # 读取重试次数
      retry_file="/tmp/retry_${{ task_id }}.txt"
      retry_count=$(cat "$retry_file" 2>/dev/null || echo 0)
      retry_count=$((retry_count + 1))
      echo $retry_count > "$retry_file"
      
      if [ $retry_count -gt 3 ]; then
        echo "Max retries exceeded"
        exit 101  # 失败退出
      fi
      
      echo "Retry attempt: $retry_count"

  - name: process-with-retry
    run: |
      # 尝试处理
      if ! process_command "${{ input_path }}" "${{ output_path }}"; then
        echo "Processing failed, can retry"
        exit 1  # 普通失败，可以重试任务
      fi
```

### 5. 依赖检查

```yaml
steps:
  - name: check-dependencies
    run: |
      # 检查输入文件的依赖
      dependency_file="${{ input_path }}.dep"
      
      if [ ! -f "$dependency_file" ]; then
        echo "Dependency file not found, cannot process"
        exit 101  # 依赖缺失，标记失败
      fi
      
      # 检查依赖是否已处理
      if ! grep -q "PROCESSED" "$dependency_file"; then
        echo "Dependencies not ready, skip for now"
        exit 100  # 暂时跳过，稍后可重新扫描
      fi
      
      echo "All dependencies satisfied"

  - name: process
    run: process_with_deps "${{ input_path }}" "${{ output_path }}"
```

## 最佳实践

### 1. 日志清晰

使用退出控制时，务必输出清晰的日志说明原因：

```bash
echo "Reason for exit: File already exists"
exit 100
```

### 2. 区分成功跳过和失败

- 使用 `exit 100`：预期的跳过情况（如文件已存在、无需处理）
- 使用 `exit 101`：错误情况（如格式不支持、文件损坏）

### 3. 第一步使用

通常在第一步或前几步使用退出控制，避免浪费资源：

```yaml
steps:
  - name: quick-check
    run: |
      # 快速检查是否需要处理
      if should_skip; then
        exit 100
      fi

  - name: expensive-operation
    run: |
      # 耗时操作只在需要时执行
      expensive_process "${{ input_path }}"
```

### 4. 结合条件判断

```bash
# 示例：根据不同条件选择不同的退出码
if [ condition1 ]; then
    echo "Success skip"
    exit 100
elif [ condition2 ]; then
    echo "Failure skip"
    exit 101
elif [ condition3 ]; then
    echo "Regular failure"
    exit 1
else
    echo "Continue processing"
    exit 0
fi
```

## 与普通失败的区别

| 特性 | 普通失败 (exit 1-99, 102-255) | 成功退出 (exit 100) | 失败退出 (exit 101) |
|-----|----------------------------|-------------------|-------------------|
| 任务状态 | failed | completed | failed |
| 步骤状态 | failed | completed | failed |
| 后续步骤 | 不执行 | 不执行 | 不执行 |
| 语义 | 处理失败 | 预期的跳过 | 预期的失败 |
| 重试建议 | 可重试 | 无需重试 | 根据原因决定 |

## 完整示例

参考 `docs/example-workflows/conditional-processing.yaml` 查看完整的示例工作流。

## 注意事项

1. **退出码必须是整数**：确保 shell 脚本返回正确的退出码
2. **日志记录**：使用 `echo` 输出说明信息，便于调试
3. **测试**：充分测试各种退出情况
4. **文档化**：在 workflow 描述中说明使用的退出控制逻辑
