# OneDrive Storage Middleware

将多个 OneDrive 账号聚合为统一存储池的中间件服务。

## 功能特性

- 🔗 **多账号聚合** - 将多个 OneDrive 账号统一管理
- 📦 **对象存储 API** - 提供标准的 S3 风格接口
- 📁 **虚拟目录** - 支持目录树结构管理
- ⚖️ **负载均衡** - 自动选择最优存储账号
- 🔄 **令牌刷新** - 自动管理 OAuth 令牌

## 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose
- Azure AD 应用凭据

### 启动服务

```bash
./start.sh
```

### 访问地址

| 功能 | 地址 |
|------|------|
| Web 界面 | http://localhost:8080/ |
| 配置指南 | http://localhost:8080/api/v1/oauth/setup |
| 添加账号 | http://localhost:8080/api/v1/oauth/create |
| 账号管理 | http://localhost:8080/api/v1/oauth/accounts |
| 健康检查 | http://localhost:8080/api/v1/health |

## API 接口

### 对象存储

```bash
# 上传文件
curl -X PUT "http://localhost:8080/api/v1/objects/{bucket}/{key}" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @file.txt

# 下载文件
curl "http://localhost:8080/api/v1/objects/{bucket}/{key}" -o file.txt

# 删除文件
curl -X DELETE "http://localhost:8080/api/v1/objects/{bucket}/{key}"

# 列出文件
curl "http://localhost:8080/api/v1/objects/{bucket}"
```

### Bucket 管理

```bash
# 创建 bucket
curl -X PUT "http://localhost:8080/api/v1/buckets/{bucket}"

# 列出 buckets
curl "http://localhost:8080/api/v1/buckets"

# 删除 bucket
curl -X DELETE "http://localhost:8080/api/v1/buckets/{bucket}"
```

### 虚拟目录

```bash
# 上传到路径
curl -X PUT "http://localhost:8080/api/v1/vfs/{bucket}/path/to/file.txt" \
  --data-binary @file.txt

# 列出目录
curl "http://localhost:8080/api/v1/vfs/{bucket}/path/"

# 创建目录
curl -X POST "http://localhost:8080/api/v1/vfs/{bucket}/path/new-folder/?mkdir"
```

## 配置

配置文件: `configs/config.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8080
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
│   ├── api/             # HTTP API
│   ├── service/         # 业务逻辑
│   ├── repository/      # 数据访问
│   ├── infrastructure/  # 外部服务
│   └── core/            # 核心组件
└── web/static/          # 前端文件
```

## License

MIT
