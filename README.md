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

### 流式传输 (HTTP Range 请求)

API 支持 HTTP Range 请求，可用于：
- **视频流播放** - 支持视频进度条拖拽
- **断点续传** - 支持大文件分段下载
- **音频流** - 支持音频进度控制

使用示例：
```bash
# 获取文件前 1024 字节
curl -H "Range: bytes=0-1023" http://localhost:8080/api/v1/objects/mybucket/video.mp4

# 获取文件从第 1MB 开始的内容
curl -H "Range: bytes=1048576-" http://localhost:8080/api/v1/objects/mybucket/video.mp4

# 获取文件最后 1MB
curl -H "Range: bytes=-1048576" http://localhost:8080/api/v1/objects/mybucket/video.mp4
```

HTML5 视频播放器会自动使用 Range 请求：
```html
<video src="http://localhost:8080/api/v1/objects/mybucket/video.mp4" controls></video>
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

// 断点续传 (Range 请求)
const partialData = await fetch('http://localhost:8080/api/v1/objects/mybucket/largefile.zip', {
  headers: { 'Range': 'bytes=0-1048575' }  // 获取第一个 1MB
});
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

## Web 云盘应用

本项目包含一个独立的 Web 云盘应用 (`web-app/`)，通过调用中间件 API 实现文件管理功能。

### 功能特性

- 📁 **文件管理** - 上传、下载、删除、重命名、移动、复制文件
- 📂 **文件夹操作** - 创建、删除、浏览文件夹
- 🔍 **文件搜索** - 快速查找文件
- 👁️ **文件预览** - 支持图片、视频、音频、文本等格式预览
- 📊 **存储统计** - 实时显示存储空间使用情况
- 🎨 **现代界面** - 响应式设计，支持网格/列表视图切换
- ⌨️ **快捷操作** - 支持拖拽上传、右键菜单、批量选择

### 启动 Web 应用

1. 首先确保中间件服务已启动：
```bash
./start.sh
```

2. 启动 Web 应用（默认端口 3000）：
```bash
./start-web.sh
```

3. 访问 Web 界面：`http://localhost:3000`

### 配置 API 地址

修改 `web-app/index.html` 中的 `API_BASE_URL` 来指定中间件 API 地址：

```html
<script>
    window.API_BASE_URL = 'http://localhost:8080/api/v1';
</script>
```

或者通过环境变量启动：
```bash
API_URL=http://your-middleware-host:8080/api/v1 ./start-web.sh
```

### 快捷键

- `Ctrl/Cmd + A` - 全选
- `Delete` - 删除选中项
- `Escape` - 取消选择或关闭弹窗

### 文件预览

支持预览以下格式：
- **图片**: jpg, jpeg, png, gif, bmp, webp, svg
- **视频**: mp4, webm, ogg
- **音频**: mp3, wav, ogg, m4a
- **文档**: txt, md, json, xml, yaml, csv, 代码文件
- **PDF**: 内嵌预览

## License

MIT
