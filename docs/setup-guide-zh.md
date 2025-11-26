# OneDrive 存储中间件 - 配置构建运行测试文档

本文档提供详细的配置、构建、运行和测试说明。

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [详细配置](#详细配置)
- [构建说明](#构建说明)
- [运行说明](#运行说明)
- [测试说明](#测试说明)
- [常见问题](#常见问题)

---

## 系统要求

### 必需组件

- **Go 1.19+**: [下载 Go](https://golang.org/dl/)
- **PostgreSQL 12+**: [下载 PostgreSQL](https://www.postgresql.org/download/)
- **Git**: [下载 Git](https://git-scm.com/downloads)

### 可选组件

- **jq**: JSON 处理工具（用于测试脚本）
- **curl**: HTTP 请求工具（用于测试）
- **golangci-lint**: 代码检查工具

---

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/xuecangming/onedrive-storage.git
cd onedrive-storage
```

### 2. 安装依赖

```bash
make deps
```

或者手动执行：

```bash
go mod download
go mod tidy
```

### 3. 配置数据库

#### 创建数据库

```bash
# 使用 psql 命令行
createdb onedrive_storage

# 或者使用 SQL
psql -U postgres
CREATE DATABASE onedrive_storage;
\q
```

#### 设置数据库密码

```bash
export DB_PASSWORD=your_password_here
```

**注意**: 生产环境请使用强密码并安全存储。

### 4. 构建项目

```bash
make build
```

这将在 `bin/` 目录下生成 `server` 可执行文件。

### 5. 运行服务

```bash
export DB_PASSWORD=your_password_here
./bin/server
```

或者使用 make 命令：

```bash
make run
```

服务将在 `http://localhost:8080` 启动。

### 6. 验证安装

```bash
# 检查健康状态
curl http://localhost:8080/api/v1/health

# 期望输出
{
  "status": "healthy",
  "timestamp": null,
  "components": {
    "database": "healthy",
    "cache": "healthy",
    "onedrive": "healthy"
  }
}
```

---

## 详细配置

### 配置文件

配置文件位于 `configs/config.yaml`。可以通过环境变量 `CONFIG_PATH` 指定自定义配置文件。

```bash
export CONFIG_PATH=/path/to/your/config.yaml
```

### 配置项说明

#### 1. 服务器配置

```yaml
server:
  host: "0.0.0.0"          # 监听地址，0.0.0.0 表示所有接口
  port: 8080               # 监听端口
  api_prefix: "/api/v1"    # API 路径前缀
```

#### 2. 数据库配置

```yaml
database:
  host: "localhost"        # 数据库主机地址
  port: 5432              # PostgreSQL 端口
  name: "onedrive_storage" # 数据库名称
  user: "postgres"        # 数据库用户名
  password: "${DB_PASSWORD}" # 数据库密码（从环境变量读取）
  max_connections: 20     # 最大连接数
```

**环境变量**:
- `DB_PASSWORD`: 数据库密码（必需）

#### 3. 缓存配置

```yaml
cache:
  enabled: false          # 是否启用缓存
  type: "memory"         # 缓存类型: memory 或 redis
  redis:
    host: "localhost"
    port: 6379
    password: "${REDIS_PASSWORD}"
    db: 0
  ttl:
    token: 3000          # 令牌缓存时间（秒）
    metadata: 300        # 元数据缓存时间（秒）
```

#### 4. 存储配置

```yaml
storage:
  upload:
    max_file_size: 107374182400    # 最大文件大小 (100GB)
    chunk_size: 10485760           # 分片大小 (10MB)
    chunk_threshold: 4194304       # 分片阈值 (4MB)
    parallel_chunks: 4             # 并行上传分片数
  
  load_balance:
    strategy: "least_used"         # 负载均衡策略
    health_check_interval: 60      # 健康检查间隔（秒）
  
  retry:
    max_attempts: 3               # 最大重试次数
    initial_delay: 1000           # 初始延迟（毫秒）
    max_delay: 30000             # 最大延迟（毫秒）
    multiplier: 2                # 延迟倍数
```

**负载均衡策略选项**:
- `least_used`: 使用率最低优先（推荐）
- `round_robin`: 轮询
- `weighted`: 加权随机

#### 5. 令牌管理配置

```yaml
token:
  refresh_before_expire: 300      # 过期前多少秒刷新（秒）
  refresh_check_interval: 60      # 检查刷新的间隔（秒）
```

#### 6. 日志配置

```yaml
logging:
  level: "info"                   # 日志级别: debug, info, warn, error
  format: "json"                  # 日志格式: json, text
  output: "stdout"                # 输出: stdout, file
  file:
    path: "/var/log/storage/app.log"
    max_size: 100                 # 单文件最大大小 (MB)
    max_backups: 10              # 最大备份数
    max_age: 30                  # 最大保存天数
```

### 环境变量

所有配置都可以通过环境变量覆盖：

```bash
# 必需
export DB_PASSWORD=your_db_password

# 可选
export REDIS_PASSWORD=your_redis_password
export CONFIG_PATH=/custom/path/config.yaml
```

---

## 构建说明

### 使用 Makefile（推荐）

```bash
# 安装依赖
make deps

# 构建项目
make build

# 清理构建产物
make clean

# 格式化代码
make fmt

# 代码检查
make vet
make lint
```

### 手动构建

```bash
# 下载依赖
go mod download

# 构建可执行文件
go build -o bin/server cmd/server/main.go

# 构建并指定输出路径
go build -o /usr/local/bin/onedrive-storage cmd/server/main.go
```

### 交叉编译

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64 cmd/server/main.go

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o bin/server-linux-arm64 cmd/server/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/server-windows-amd64.exe cmd/server/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/server-darwin-amd64 cmd/server/main.go
```

### 构建优化

```bash
# 生产环境构建（优化大小和性能）
go build -ldflags="-s -w" -o bin/server cmd/server/main.go

# 添加版本信息
VERSION=$(git describe --tags --always)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
go build -ldflags="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
  -o bin/server cmd/server/main.go
```

---

## 运行说明

### 基本运行

```bash
# 方式1: 使用 make
make run

# 方式2: 直接运行可执行文件
export DB_PASSWORD=your_password
./bin/server

# 方式3: 使用 go run
go run cmd/server/main.go
```

### 后台运行

```bash
# 使用 nohup
nohup ./bin/server > server.log 2>&1 &

# 查看进程
ps aux | grep server

# 停止服务
pkill -f server
# 或者
kill $(pgrep -f server)
```

### 使用 systemd（Linux 生产环境推荐）

创建服务文件 `/etc/systemd/system/onedrive-storage.service`:

```ini
[Unit]
Description=OneDrive Storage Middleware
After=network.target postgresql.service

[Service]
Type=simple
User=storage
Group=storage
WorkingDirectory=/opt/onedrive-storage
Environment="DB_PASSWORD=your_password"
ExecStart=/opt/onedrive-storage/bin/server
Restart=always
RestartSec=10

# 日志
StandardOutput=journal
StandardError=journal

# 安全设置
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

管理服务：

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start onedrive-storage

# 停止服务
sudo systemctl stop onedrive-storage

# 重启服务
sudo systemctl restart onedrive-storage

# 查看状态
sudo systemctl status onedrive-storage

# 开机自启
sudo systemctl enable onedrive-storage

# 查看日志
sudo journalctl -u onedrive-storage -f
```

### Docker 运行（待实现）

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run

# 或者使用 docker-compose
docker-compose -f docker/docker-compose.yaml up -d
```

### 验证服务运行

```bash
# 检查健康状态
curl http://localhost:8080/api/v1/health

# 检查服务信息
curl http://localhost:8080/api/v1/info

# 预期输出
{
  "name": "OneDrive Storage Middleware",
  "version": "1.0.0",
  "api_version": "v1"
}
```

---

## 测试说明

### 自动化测试

项目包含两个测试脚本：

#### 1. 对象存储 API 测试

```bash
./scripts/test_api.sh
```

**测试内容**（20 个测试）:
- 系统健康检查
- Bucket 管理（创建、列表、删除）
- 对象存储（上传、下载、元数据、列表、删除）
- 分页功能
- 错误处理

**输出示例**:
```
========================================
Object Storage API Test Suite
========================================

✓ PASS: Health check endpoint
✓ PASS: Service info endpoint
✓ PASS: List buckets (empty state)
...

Total tests run: 20
Passed: 20
Failed: 0

All tests passed!
```

#### 2. 虚拟文件系统测试

```bash
./scripts/test_vfs.sh
```

**测试内容**（15 个测试）:
- 上传文件到虚拟路径
- 列出目录（普通和递归）
- 下载文件
- 移动/重命名文件
- 删除文件和目录（递归）
- 自动创建父目录
- 错误处理

**输出示例**:
```
========================================
VFS API Test Suite
========================================

✓ PASS: Upload file to virtual path
✓ PASS: List root directory
✓ PASS: List directory recursively
...

Total tests run: 15
Passed: 15
Failed: 0

All tests passed!
```

### 单元测试

```bash
# 运行所有单元测试
go test ./...

# 运行特定包的测试
go test ./internal/service/object/...

# 详细输出
go test -v ./...

# 显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 集成测试

**前提条件**:
1. PostgreSQL 数据库运行中
2. 设置 `DB_PASSWORD` 环境变量
3. 服务器正在运行

```bash
# 启动服务器
export DB_PASSWORD=postgres
./bin/server &
SERVER_PID=$!

# 等待服务器启动
sleep 3

# 运行测试
./scripts/test_api.sh
./scripts/test_vfs.sh

# 停止服务器
kill $SERVER_PID
```

### 手动测试

#### 1. 对象存储 API

```bash
# 创建 Bucket
curl -X PUT http://localhost:8080/api/v1/buckets/test-bucket

# 上传文件
echo "Hello World" > test.txt
curl -X PUT http://localhost:8080/api/v1/objects/test-bucket/test.txt \
  -H "Content-Type: text/plain" \
  --data-binary @test.txt

# 下载文件
curl http://localhost:8080/api/v1/objects/test-bucket/test.txt

# 列出对象
curl http://localhost:8080/api/v1/objects/test-bucket

# 获取元数据
curl -I http://localhost:8080/api/v1/objects/test-bucket/test.txt

# 删除对象
curl -X DELETE http://localhost:8080/api/v1/objects/test-bucket/test.txt

# 删除 Bucket
curl -X DELETE http://localhost:8080/api/v1/buckets/test-bucket
```

#### 2. 虚拟文件系统 API

```bash
# 创建 Bucket
curl -X PUT http://localhost:8080/api/v1/buckets/vfs-bucket

# 上传到虚拟路径（自动创建目录）
echo "Report 2024" > report.txt
curl -X PUT http://localhost:8080/api/v1/vfs/vfs-bucket/docs/2024/report.txt \
  -H "Content-Type: text/plain" \
  --data-binary @report.txt

# 列出根目录
curl "http://localhost:8080/api/v1/vfs/vfs-bucket/?type=directory"

# 列出特定目录
curl "http://localhost:8080/api/v1/vfs/vfs-bucket/docs/?type=directory"

# 递归列出目录
curl "http://localhost:8080/api/v1/vfs/vfs-bucket/docs/?type=directory&recursive=true"

# 下载文件
curl http://localhost:8080/api/v1/vfs/vfs-bucket/docs/2024/report.txt

# 移动/重命名文件
curl -X POST http://localhost:8080/api/v1/vfs/vfs-bucket/_move \
  -H "Content-Type: application/json" \
  -d '{
    "source": "/docs/2024/report.txt",
    "destination": "/archive/report-2024.txt"
  }'

# 创建空目录
curl -X POST "http://localhost:8080/api/v1/vfs/vfs-bucket/_mkdir?path=/photos/"

# 删除文件
curl -X DELETE http://localhost:8080/api/v1/vfs/vfs-bucket/archive/report-2024.txt

# 递归删除目录
curl -X DELETE "http://localhost:8080/api/v1/vfs/vfs-bucket/archive/?type=directory&recursive=true"
```

#### 3. 空间管理 API

```bash
# 查看空间概览
curl http://localhost:8080/api/v1/space

# 列出所有存储账号
curl http://localhost:8080/api/v1/space/accounts

# 查看特定账号详情
curl http://localhost:8080/api/v1/space/accounts/{account_id}

# 同步账号空间信息
curl -X POST http://localhost:8080/api/v1/space/accounts/{account_id}/sync
```

### 性能测试

使用 Apache Bench (ab) 进行压力测试：

```bash
# 安装 ab
sudo apt-get install apache2-utils  # Ubuntu/Debian
brew install apache2-utils           # macOS

# 并发上传测试
ab -n 1000 -c 10 -p test.txt -T text/plain \
  http://localhost:8080/api/v1/objects/test-bucket/perf-test.txt

# 下载性能测试
ab -n 1000 -c 10 \
  http://localhost:8080/api/v1/objects/test-bucket/test.txt
```

使用 wrk 进行高级压测：

```bash
# 安装 wrk
sudo apt-get install wrk  # Ubuntu/Debian
brew install wrk          # macOS

# GET 请求压测
wrk -t4 -c100 -d30s http://localhost:8080/api/v1/health

# 自定义脚本
wrk -t4 -c100 -d30s -s upload.lua http://localhost:8080/api/v1/objects/test-bucket/
```

---

## 常见问题

### 1. 数据库连接失败

**问题**: `Failed to connect to database: dial tcp [::1]:5432: connect: connection refused`

**解决方案**:
```bash
# 检查 PostgreSQL 是否运行
sudo systemctl status postgresql

# 启动 PostgreSQL
sudo systemctl start postgresql

# 检查数据库是否存在
psql -U postgres -l | grep onedrive_storage

# 创建数据库
createdb -U postgres onedrive_storage

# 测试连接
psql -U postgres -d onedrive_storage -c "SELECT 1;"
```

### 2. 端口已被占用

**问题**: `bind: address already in use`

**解决方案**:
```bash
# 查找占用端口的进程
lsof -i :8080
# 或
netstat -tlnp | grep 8080

# 停止占用端口的进程
kill -9 <PID>

# 或者修改配置文件使用其他端口
# configs/config.yaml
server:
  port: 8081
```

### 3. 权限错误

**问题**: `permission denied`

**解决方案**:
```bash
# 给可执行文件添加执行权限
chmod +x bin/server
chmod +x scripts/*.sh

# 检查数据库权限
psql -U postgres
GRANT ALL PRIVILEGES ON DATABASE onedrive_storage TO postgres;
```

### 4. 依赖下载失败

**问题**: `go: module xxx: Get "https://proxy.golang.org/...": dial tcp: i/o timeout`

**解决方案**:
```bash
# 设置 Go 代理（中国用户）
export GOPROXY=https://goproxy.cn,direct

# 或使用阿里云代理
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 永久设置
echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc
source ~/.bashrc

# 清除缓存重新下载
go clean -modcache
go mod download
```

### 5. 测试脚本失败

**问题**: 测试脚本报错或超时

**解决方案**:
```bash
# 确保服务器运行
ps aux | grep server

# 检查服务器健康状态
curl http://localhost:8080/api/v1/health

# 清理测试数据
curl -X DELETE http://localhost:8080/api/v1/buckets/test-bucket
curl -X DELETE http://localhost:8080/api/v1/buckets/vfs-test-bucket

# 重启服务器
pkill -f server
sleep 2
export DB_PASSWORD=postgres
./bin/server &
sleep 3

# 重新运行测试
./scripts/test_api.sh
./scripts/test_vfs.sh
```

### 6. 数据库迁移问题

**问题**: 表已存在或迁移失败

**解决方案**:
```bash
# 查看现有表
psql -U postgres -d onedrive_storage -c "\dt"

# 删除所有表（谨慎！生产环境不要这样做）
psql -U postgres -d onedrive_storage << EOF
DROP TABLE IF EXISTS virtual_files CASCADE;
DROP TABLE IF EXISTS virtual_directories CASCADE;
DROP TABLE IF EXISTS object_chunks CASCADE;
DROP TABLE IF EXISTS objects CASCADE;
DROP TABLE IF EXISTS buckets CASCADE;
DROP TABLE IF EXISTS storage_accounts CASCADE;
EOF

# 重启服务器会自动重新创建表
```

### 7. 内存不足

**问题**: 上传大文件时内存不足

**解决方案**:
```bash
# 检查配置中的文件大小限制
# configs/config.yaml
storage:
  upload:
    max_file_size: 10737418240  # 减小到 10GB
    chunk_size: 5242880         # 减小到 5MB
    chunk_threshold: 2097152    # 减小到 2MB

# 增加系统可用内存
# 或使用分片上传
```

### 8. 日志查看

```bash
# 实时查看日志（如果输出到文件）
tail -f /var/log/storage/app.log

# 使用 systemd
sudo journalctl -u onedrive-storage -f

# 查看最近的错误
sudo journalctl -u onedrive-storage -p err

# 查看特定时间范围
sudo journalctl -u onedrive-storage --since "1 hour ago"
```

---

## 开发提示

### 代码格式化

```bash
# 格式化所有代码
make fmt

# 或
go fmt ./...

# 使用 goimports（自动处理 import）
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

### 代码检查

```bash
# 运行 go vet
make vet

# 运行 golangci-lint
make lint

# 手动安装 golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### 调试

```bash
# 使用 delve 调试器
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug cmd/server/main.go

# 或附加到运行中的进程
dlv attach $(pgrep server)
```

---

## 生产部署建议

1. **使用反向代理**: 在生产环境使用 Nginx 或 Traefik
2. **HTTPS**: 配置 SSL/TLS 证书
3. **监控**: 集成 Prometheus + Grafana
4. **日志**: 使用 ELK 或 Loki 收集日志
5. **备份**: 定期备份 PostgreSQL 数据库
6. **限流**: 在反向代理层配置限流
7. **安全**: 配置防火墙规则，限制访问

---

## 更多文档

- [API 文档](api.md) - 完整的 API 接口说明
- [快速开始](quickstart.md) - 简化版快速开始指南
- [开发进度](progress.md) - 项目开发进度
- [Phase 3 总结](phase3-summary.md) - Phase 3 实现详情
- [Phase 4 总结](phase4-summary.md) - Phase 4 VFS 实现详情

---

## 获取帮助

如果遇到问题：
1. 查看本文档的常见问题部分
2. 检查服务器日志
3. 在 GitHub 上提交 Issue
4. 参考项目 README.md

---

*文档版本: 1.0*
*最后更新: Phase 4 完成*
