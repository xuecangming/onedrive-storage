# OneDrive Storage Middleware

将多个 OneDrive 账号聚合为统一存储池的 API 中间件服务。

## 功能特性

- 🔗 **多账号聚合** - 将多个 OneDrive 账号统一管理
- 📦 **对象存储 API** - 提供标准的 RESTful 接口
- 📁 **虚拟目录** - 支持目录树结构管理
- ⚖️ **负载均衡** - 自动选择最优存储账号
- 🔄 **令牌刷新** - 自动管理 OAuth 令牌
- 🌐 **CORS 支持** - 支持前端跨域访问

## 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose
- Azure AD 应用凭据

### 启动服务

```bash
./start.sh
```

服务启动后，API 地址: `http://localhost:8080/api/v1`

## API 接口

### 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

### Bucket 管理

```bash
# 列出 buckets
curl http://localhost:8080/api/v1/buckets

# 创建 bucket
curl -X PUT http://localhost:8080/api/v1/buckets/{bucket}

# 删除 bucket
curl -X DELETE http://localhost:8080/api/v1/buckets/{bucket}
```

### 对象存储

```bash
# 上传文件
curl -X PUT http://localhost:8080/api/v1/objects/{bucket}/{key} \
  -H "Content-Type: application/octet-stream" \
  --data-binary @file.txt

# 下载文件
curl http://localhost:8080/api/v1/objects/{bucket}/{key} -o file.txt

# 删除文件
curl -X DELETE http://localhost:8080/api/v1/objects/{bucket}/{key}

# 列出文件
curl http://localhost:8080/api/v1/objects/{bucket}
```

### 虚拟目录 (VFS)

```bash
# 上传到路径
curl -X PUT http://localhost:8080/api/v1/vfs/{bucket}/path/to/file.txt \
  --data-binary @file.txt

# 列出目录
curl http://localhost:8080/api/v1/vfs/{bucket}/path/

# 创建目录
curl -X POST http://localhost:8080/api/v1/vfs/{bucket}/_mkdir \
  -d '{"path": "/new-folder"}'
```

### 账号管理

```bash
# 列出账号
curl http://localhost:8080/api/v1/accounts

# 创建账号
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name":"账号1","email":"user@example.com","client_id":"...","client_secret":"...","tenant_id":"..."}'

# 同步空间信息
curl -X POST http://localhost:8080/api/v1/accounts/{id}/sync
```

### OAuth 授权

```bash
# 获取配置指南 (HTML)
curl http://localhost:8080/api/v1/oauth/setup

# 创建账号页面 (HTML)
curl http://localhost:8080/api/v1/oauth/create

# 发起授权
curl http://localhost:8080/api/v1/oauth/authorize/{id}
```

### 空间统计

```bash
# 获取空间概览
curl http://localhost:8080/api/v1/space

# 列出账号空间
curl http://localhost:8080/api/v1/space/accounts
```

## 配置

配置文件: `configs/config.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  api_prefix: "/api/v1"
  base_url: ""  # OAuth 回调 URL，留空则自动检测

database:
  host: "localhost"
  port: 5432
  name: "onedrive_storage"
  user: "postgres"
  password: "postgres123"
```

## 项目结构

```
├── cmd/server/          # 程序入口
├── configs/             # 配置文件
├── internal/
│   ├── api/             # HTTP API 层
│   │   ├── handlers/    # 请求处理器
│   │   └── middleware/  # 中间件 (CORS, 日志, 恢复)
│   ├── service/         # 业务逻辑层
│   ├── repository/      # 数据访问层
│   ├── infrastructure/  # 外部服务 (OneDrive, 数据库)
│   └── core/            # 核心组件 (负载均衡, 重试)
└── scripts/             # 测试脚本
```

## 前端接入

中间件已启用 CORS，支持任意前端应用跨域访问。

### CORS 配置

默认允许所有来源。生产环境中，建议通过环境变量限制允许的来源：

```bash
# 限制 CORS 来源
export CORS_ALLOWED_ORIGINS="https://your-domain.com, https://app.your-domain.com"
```

### 速率限制

中间件支持 IP 级别的速率限制。在 API 路由中启用：

```go
import "github.com/xuecangming/onedrive-storage/internal/api/middleware"

// 每秒允许 100 个请求
router.Use(middleware.RateLimitMiddleware(100, time.Second))
```

示例 (JavaScript):
```javascript
// 上传文件
const response = await fetch('http://localhost:8080/api/v1/objects/mybucket/test.txt', {
  method: 'PUT',
  body: fileContent
});

// 下载文件
const data = await fetch('http://localhost:8080/api/v1/objects/mybucket/test.txt');
```

## 测试

运行单元测试：

```bash
go test ./...
```

运行 API 测试：

```bash
./scripts/test_api.sh
./scripts/test_vfs.sh
```

## License

MIT
