# SQLite to MySQL Migration Guide

## 已完成的更改

### 1. 代码更改

#### backend/database/db.go
- ✅ 替换 `modernc.org/sqlite` 为 `github.com/go-sql-driver/mysql`
- ✅ 移除 SQLite 特定的 PRAGMA 语句
- ✅ 更新连接池设置以适应 MySQL
- ✅ 将 schema 从 SQLite 语法转换为 MySQL 语法：
  - `TEXT` → `VARCHAR(n)` / `TEXT` / `MEDIUMTEXT`
  - `INTEGER` → `TINYINT(1)` / `INT` / `BIGINT`
  - `DATETIME` 使用 MySQL 的 `DEFAULT CURRENT_TIMESTAMP` 和 `ON UPDATE CURRENT_TIMESTAMP`
  - 添加 `ENGINE=InnoDB` 和 `CHARSET=utf8mb4`

#### backend/database/task_repo.go 和 task_step_repo.go
- ✅ 移除 SQLite 锁定重试逻辑（MySQL 不需要）

#### main.go
- ✅ 添加注释说明 DSN 格式

### 2. Docker Compose 更改

#### docker-compose.yml
- ✅ 添加 MySQL 8.0 服务
- ✅ 配置数据库：
  - 数据库名：`fileaction`
  - 用户名：`fileaction`
  - 密码：`fileaction_pass`
- ✅ 添加健康检查确保 MySQL 启动后再启动应用
- ✅ 更新 fileaction 服务的 `DB_PATH` 环境变量为 MySQL DSN
- ✅ 添加持久化卷：
  - `mysql_data`: MySQL 数据
  - `fileaction_logs`: 应用日志

### 3. 配置文件更改

#### config/config.yaml
- ✅ 更新 `database.path` 为 MySQL DSN 格式
- ✅ 添加注释说明 SQLite 和 MySQL 配置

### 4. 依赖管理
- ✅ 添加 `github.com/go-sql-driver/mysql` v1.9.3
- ✅ 更新 vendor 目录

## MySQL DSN 格式

```
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

### 示例：

**Docker Compose 内部连接：**
```
fileaction:fileaction_pass@tcp(mysql:3306)/fileaction?charset=utf8mb4&parseTime=True&loc=Local
```

**本地开发连接：**
```
fileaction:fileaction_pass@tcp(localhost:3306)/fileaction?charset=utf8mb4&parseTime=True&loc=Local
```

## 使用方法

### 1. 启动服务（Docker Compose）

```bash
docker-compose up -d
```

这将：
1. 启动 MySQL 服务并等待健康检查通过
2. 自动创建数据库和用户
3. 启动 FileAction 应用
4. 应用自动创建所有必要的表

### 2. 本地开发

如果要在本地运行（不使用 Docker）：

1. 启动 MySQL 服务
2. 创建数据库和用户：
```sql
CREATE DATABASE fileaction CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'fileaction'@'localhost' IDENTIFIED BY 'fileaction_pass';
GRANT ALL PRIVILEGES ON fileaction.* TO 'fileaction'@'localhost';
FLUSH PRIVILEGES;
```

3. 更新 `config/config.yaml`：
```yaml
database:
  path: "fileaction:fileaction_pass@tcp(localhost:3306)/fileaction?charset=utf8mb4&parseTime=True&loc=Local"
```

4. 运行应用：
```bash
./fileaction
```

### 3. 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 仅查看应用日志
docker-compose logs -f fileaction

# 仅查看 MySQL 日志
docker-compose logs -f mysql
```

### 4. 访问 MySQL

```bash
# 从容器外部
mysql -h 127.0.0.1 -P 3306 -u fileaction -pfileaction_pass fileaction

# 从容器内部
docker exec -it fileaction-mysql mysql -u fileaction -pfileaction_pass fileaction
```

## 数据迁移（可选）

如果需要从现有的 SQLite 数据库迁移数据：

1. 导出 SQLite 数据为 SQL：
```bash
sqlite3 data/fileaction.db .dump > sqlite_dump.sql
```

2. 转换 SQL 语法（需要手动或使用工具）：
   - 移除 SQLite 特定语法
   - 调整数据类型
   - 修改时间戳格式

3. 导入到 MySQL：
```bash
mysql -h 127.0.0.1 -u fileaction -pfileaction_pass fileaction < converted_dump.sql
```

## 性能优化

MySQL 配置已针对并发优化：
- ✅ 连接池：最大 25 个开放连接，5 个空闲连接
- ✅ 使用 InnoDB 引擎支持事务和外键
- ✅ UTF8MB4 字符集支持完整的 Unicode
- ✅ 索引优化用于常见查询

## 故障排查

### 问题：无法连接到 MySQL
- 检查 MySQL 容器是否运行：`docker-compose ps`
- 检查健康检查状态：`docker-compose ps mysql`
- 查看 MySQL 日志：`docker-compose logs mysql`

### 问题：权限错误
- 确保 MySQL 用户有正确的权限
- 检查 DSN 中的用户名和密码是否正确

### 问题：字符编码问题
- 确保 DSN 中包含 `charset=utf8mb4`
- 检查表的字符集：`SHOW CREATE TABLE workflows;`

## 优势总结

相比 SQLite，MySQL 提供：
- ✅ 更好的并发性能（无数据库锁定问题）
- ✅ 更大的数据容量支持
- ✅ 更好的事务处理
- ✅ 生产环境标准选择
- ✅ 更丰富的监控和管理工具
