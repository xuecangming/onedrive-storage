好的！下面是完整的方案详细描述。

---

# OneDrive 统一存储中间件 - 详细设计方案

## 一、项目概述

### 1.1 项目定位

本项目是一个**面向上层应用的统一存储中间件**，将多个 Microsoft 365 E3 开发者账号的 OneDrive 存储空间聚合为统一的存储池，对外提供标准化的存储服务接口。

### 1.2 核心目标

```
┌─────────────────────────────────────────────────────────────┐
│                        核心目标                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 空间聚合    将 25 个 OneDrive 账号聚合为统一存储池         │
│                 对上层应用透明，无需感知多账号                  │
│                                                              │
│  2. 接口标准    提供简洁、通用的对象存储接口                    │
│                 降低上层应用的接入成本                         │
│                                                              │
│  3. 可靠服务    自动处理令牌刷新、故障转移、重试等              │
│                 保障存储服务的稳定性                           │
│                                                              │
│  4. 灵活扩展    支持虚拟目录等可选功能层                       │
│                 满足不同类型应用的需求                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 系统架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           应用层                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  网盘应用    │  │  图床应用    │  │  备份应用    │  │  其他应用   │     │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│         │                │                │                │            │
│         │    使用虚拟目录  │    直接使用     │    按需选择     │            │
│         ▼                ▼                ▼                ▼            │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     OneDrive Storage Middleware                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                      API 网关层 (API Gateway)                    │   │
│   │   • 请求路由    • 认证鉴权    • 限流控制    • 请求日志            │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│                  ┌─────────────────┴─────────────────┐                  │
│                  ▼                                   ▼                  │
│   ┌──────────────────────────────┐   ┌──────────────────────────────┐   │
│   │   虚拟目录服务 (可选层)        │   │   对象存储服务 (核心层)        │   │
│   │   Virtual Directory Service  │   │   Object Storage Service     │   │
│   │                              │   │                              │   │
│   │   • 目录树管理               │   │   • 对象上传/下载              │   │
│   │   • 路径解析                 │──▶│   • 对象删除                  │   │
│   │   • 移动/重命名              │   │   • 对象列表                  │   │
│   │   • 文件夹操作               │   │   • Bucket 管理               │   │
│   └──────────────────────────────┘   └──────────────┬───────────────┘   │
│                                                      │                   │
│   ┌──────────────────────────────────────────────────┴───────────────┐   │
│   │                      基础设施层 (Infrastructure)                  │   │
│   │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐    │   │
│   │  │ 账号管理    │ │ 空间调度    │ │ 分片处理    │ │ 任务队列    │    │   │
│   │  │ Account    │ │ Space      │ │ Chunker    │ │ Task       │    │   │
│   │  │ Manager    │ │ Scheduler  │ │            │ │ Queue      │    │   │
│   │  └────────────┘ └────────────┘ └────────────┘ └────────────┘    │   │
│   │  ┌────────────┐ ┌────────────┐ ┌────────────┐                   │   │
│   │  │ OneDrive   │ │ 元数据     │ │ 缓存       │                   │   │
│   │  │ Client     │ │ Database   │ │ Cache      │                   │   │
│   │  └────────────┘ └────────────┘ └────────────┘                   │   │
│   └──────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
└────────────────────────────────────┼─────────────────────────────────────┘
                                     ▼
                  ┌─────────────────────────────────────┐
                  │       OneDrive Accounts Pool        │
                  │   ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐  │
                  │   │ E3-1│ │ E3-2│ │ E3-3│ │ ...  │  │
                  │   │ 1TB │ │ 1TB │ │ 1TB │ │     │  │
                  │   └─────┘ └─────┘ └─────┘ └─────┘  │
                  │           Total: ~25TB             │
                  └─────────────────────────────────────┘
```

---

## 二、核心概念定义

### 2.1 存储模型

```
┌─────────────────────────────────────────────────────────────┐
│                      存储模型                                │
└─────────────────────────────────────────────────────────────┘

┌─────────────┐
│   Bucket    │  桶：顶层存储容器，用于隔离不同应用或业务
│   (桶)      │  • 全局唯一命名
└──────┬──────┘  • 例：images、documents、backups
       │
       │ 包含多个
       ▼
┌─────────────┐
│   Object    │  对象：存储的基本单位
│   (对象)    │  • 由 Key 唯一标识
└──────┬──────┘  • 包含数据内容和元信息
       │
       │ 对象属性
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Key          对象键，在 Bucket 内唯一，例：a1b2c3d4e5f6     │
│  Size         对象大小（字节）                               │
│  ETag         内容哈希，用于校验                             │
│  MimeType     MIME 类型，如 image/jpeg                      │
│  Metadata     自定义元数据（可选）                           │
│  CreatedAt    创建时间                                      │
│  UpdatedAt    更新时间                                      │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 虚拟目录模型（可选层）

```
┌─────────────────────────────────────────────────────────────┐
│                    虚拟目录模型                              │
└─────────────────────────────────────────────────────────────┘

虚拟目录层在对象存储之上构建树形目录结构：

                    Bucket: photos
                         │
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
        /2024/        /2023/      /favorites/
           │
     ┌─────┴─────┐
     ▼           ▼
 /vacation/   /work/
     │
  ┌──┴──┐
  ▼     ▼
beach.jpg  sunset.jpg
  │
  │ 映射到
  ▼
┌─────────────────────────────────────────────────────────────┐
│  Object                                                      │
│  Bucket: photos                                              │
│  Key: obj_f8a3b2c1d4e5  ← 实际存储的对象 Key                  │
└─────────────────────────────────────────────────────────────┘

关键点：
• 虚拟目录和虚拟文件的信息存储在数据库中
• 实际数据存储在对象存储层
• 移动/重命名只修改数据库，不移动实际数据
• 一个对象可以被多个虚拟路径引用（未来可支持）
```

---

## 三、功能模块详细设计

### 3.1 对象存储服务（核心层）

#### 3.1. 1 功能清单

| 功能 | 接口 | 描述 | 优先级 |
|-----|------|------|-------|
| 上传对象 | `PUT /objects/{bucket}/{key}` | 上传单个对象 | P0 |
| 下载对象 | `GET /objects/{bucket}/{key}` | 下载单个对象 | P0 |
| 删除对象 | `DELETE /objects/{bucket}/{key}` | 删除单个对象 | P0 |
| 查询元信息 | `HEAD /objects/{bucket}/{key}` | 获取对象元信息 | P0 |
| 列出对象 | `GET /objects/{bucket}` | 列出桶内对象 | P0 |
| 检查存在 | `HEAD /objects/{bucket}/{key}` | 检查对象是否存在 | P0 |
| 创建桶 | `PUT /buckets/{bucket}` | 创建新的桶 | P0 |
| 删除桶 | `DELETE /buckets/{bucket}` | 删除空桶 | P1 |
| 列出桶 | `GET /buckets` | 列出所有桶 | P0 |

#### 3.1.2 接口详细定义

##### 上传对象

```yaml
PUT /api/v1/objects/{bucket}/{key}

描述: 上传对象到指定桶

路径参数:
  bucket: 桶名称 (3-63字符, 小写字母/数字/连字符)
  key: 对象键 (1-1024字符)

请求头:
  Content-Type: 文件的 MIME 类型
  Content-Length: 文件大小（字节）
  X-Meta-*: 自定义元数据（可选）

请求体:
  文件二进制内容

响应 (200 OK):
  {
    "bucket": "images",
    "key": "a1b2c3d4e5f6",
    "size": 2048576,
    "etag": "d41d8cd98f00b204e9800998ecf8427e",
    "mime_type": "image/jpeg",
    "created_at": "2024-01-15T10:30:00Z"
  }

错误响应:
  400 Bad Request     - 参数无效
  409 Conflict        - 对象已存在（如果设置不覆盖）
  413 Payload Too Large - 文件过大
  507 Insufficient Storage - 存储空间不足
```

##### 下载对象

```yaml
GET /api/v1/objects/{bucket}/{key}

描述: 下载指定对象

路径参数:
  bucket: 桶名称
  key: 对象键

请求头 (可选):
  Range: bytes=0-1023  (支持断点续传)

响应 (200 OK / 206 Partial Content):
  响应头:
    Content-Type: image/jpeg
    Content-Length: 2048576
    ETag: "d41d8cd98f00b204e9800998ecf8427e"
    Accept-Ranges: bytes
  响应体:
    文件二进制内容

错误响应:
  404 Not Found - 对象不存在
```

##### 删除对象

```yaml
DELETE /api/v1/objects/{bucket}/{key}

描述: 删除指定对象

路径参数:
  bucket: 桶名称
  key: 对象键

响应 (204 No Content):
  无响应体

错误响应:
  404 Not Found - 对象不存在
```

##### 查询对象元信息

```yaml
HEAD /api/v1/objects/{bucket}/{key}

描述: 获取对象元信息，不返回内容

响应 (200 OK):
  响应头:
    Content-Type: image/jpeg
    Content-Length: 2048576
    ETag: "d41d8cd98f00b204e9800998ecf8427e"
    Last-Modified: Mon, 15 Jan 2024 10:30:00 GMT
    X-Meta-*: 自定义元数据

错误响应:
  404 Not Found - 对象不存在
```

##### 列出对象

```yaml
GET /api/v1/objects/{bucket}

描述: 列出桶内的对象

路径参数:
  bucket: 桶名称

查询参数:
  prefix: 前缀过滤 (可选)
  marker: 分页游标 (可选)
  max_keys: 最大返回数量, 默认 1000, 最大 1000 (可选)

响应 (200 OK):
  {
    "bucket": "images",
    "prefix": "photos/",
    "objects": [
      {
        "key": "photos/a1b2c3",
        "size": 2048576,
        "etag": "d41d8cd98f00b204e9800998ecf8427e",
        "last_modified": "2024-01-15T10:30:00Z"
      },
      ... 
    ],
    "is_truncated": true,
    "next_marker": "photos/x9y8z7"
  }
```

##### Bucket 管理

```yaml
# 列出所有桶
GET /api/v1/buckets

响应 (200 OK):
  {
    "buckets": [
      {
        "name": "images",
        "created_at": "2024-01-01T00:00:00Z",
        "object_count": 1500,
        "total_size": 5368709120
      },
      ...
    ]
  }

# 创建桶
PUT /api/v1/buckets/{bucket}

响应 (201 Created):
  {
    "name": "images",
    "created_at": "2024-01-15T10:30:00Z"
  }

错误响应:
  409 Conflict - 桶已存在

# 删除桶
DELETE /api/v1/buckets/{bucket}

响应 (204 No Content)

错误响应:
  404 Not Found - 桶不存在
  409 Conflict - 桶非空，无法删除
```

---

### 3. 2 虚拟目录服务（可选层）

#### 3. 2.1 功能清单

| 功能 | 接口 | 描述 | 优先级 |
|-----|------|------|-------|
| 上传到路径 | `PUT /vfs/{bucket}/{path}` | 上传文件到指定路径 | P0 |
| 获取/列表 | `GET /vfs/{bucket}/{path}` | 获取文件或列出目录 | P0 |
| 删除 | `DELETE /vfs/{bucket}/{path}` | 删除文件或目录 | P0 |
| 创建目录 | `POST /vfs/{bucket}/{path}? mkdir` | 创建空目录 | P1 |
| 移动/重命名 | `POST /vfs/{bucket}/_move` | 移动或重命名 | P1 |
| 复制 | `POST /vfs/{bucket}/_copy` | 复制文件或目录 | P2 |
| 获取元信息 | `HEAD /vfs/{bucket}/{path}` | 获取文件元信息 | P1 |

#### 3.2.2 接口详细定义

##### 上传文件到路径

```yaml
PUT /api/v1/vfs/{bucket}/{path}

描述: 上传文件到指定虚拟路径，自动创建中间目录

路径参数:
  bucket: 桶名称
  path: 虚拟路径，如 documents/2024/report.pdf

请求头:
  Content-Type: 文件的 MIME 类型
  Content-Length: 文件大小

请求体:
  文件二进制内容

响应 (201 Created):
  {
    "id": "vf_a1b2c3d4e5f6",
    "name": "report.pdf",
    "path": "/documents/2024/report.pdf",
    "type": "file",
    "size": 2048576,
    "mime_type": "application/pdf",
    "object_key": "obj_x1y2z3",
    "created_at": "2024-01-15T10:30:00Z"
  }

错误响应:
  409 Conflict - 该路径已存在文件
```

##### 获取文件或列出目录

```yaml
GET /api/v1/vfs/{bucket}/{path}

描述: 
  - 如果路径是文件：下载文件内容
  - 如果路径是目录：列出目录内容

路径参数:
  bucket: 桶名称
  path: 虚拟路径（末尾带 / 表示目录）

查询参数（目录列表时）:
  recursive: 是否递归列出，默认 false
  page: 页码，默认 1
  page_size: 每页数量，默认 100

响应 - 文件 (200 OK):
  文件二进制内容（同对象下载）

响应 - 目录 (200 OK):
  {
    "path": "/documents/2024/",
    "items": [
      {
        "id": "vd_a1b2c3",
        "name": "reports",
        "path": "/documents/2024/reports/",
        "type": "directory",
        "created_at": "2024-01-10T08:00:00Z"
      },
      {
        "id": "vf_d4e5f6",
        "name": "summary.pdf",
        "path": "/documents/2024/summary.pdf",
        "type": "file",
        "size": 1024000,
        "mime_type": "application/pdf",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 25,
    "page": 1,
    "page_size": 100
  }

错误响应:
  404 Not Found - 路径不存在
```

##### 删除文件或目录

```yaml
DELETE /api/v1/vfs/{bucket}/{path}

描述: 删除指定路径的文件或目录

路径参数:
  bucket: 桶名称
  path: 虚拟路径

查询参数:
  recursive: 删除目录时是否递归删除内容，默认 false

响应 (204 No Content):
  无响应体

错误响应:
  404 Not Found - 路径不存在
  409 Conflict - 目录非空且未设置 recursive=true
```

##### 创建目录

```yaml
POST /api/v1/vfs/{bucket}/{path}?mkdir

描述: 创建空目录，自动创建中间目录

路径参数:
  bucket: 桶名称
  path: 目录路径（末尾建议带 /）

响应 (201 Created):
  {
    "id": "vd_a1b2c3d4e5f6",
    "name": "new-folder",
    "path": "/documents/new-folder/",
    "type": "directory",
    "created_at": "2024-01-15T10:30:00Z"
  }

错误响应:
  409 Conflict - 目录已存在
```

##### 移动/重命名

```yaml
POST /api/v1/vfs/{bucket}/_move

描述: 移动或重命名文件/目录

请求体:
  {
    "source": "/documents/old-name.pdf",
    "destination": "/archive/new-name. pdf"
  }

响应 (200 OK):
  {
    "id": "vf_a1b2c3d4e5f6",
    "name": "new-name. pdf",
    "path": "/archive/new-name.pdf",
    "type": "file",
    "size": 2048576
  }

错误响应:
  404 Not Found - 源路径不存在
  409 Conflict - 目标路径已存在
```

##### 复制

```yaml
POST /api/v1/vfs/{bucket}/_copy

描述: 复制文件/目录到新位置

请求体:
  {
    "source": "/documents/report.pdf",
    "destination": "/backup/report. pdf"
  }

响应 (201 Created):
  {
    "id": "vf_new123456",
    "name": "report.pdf",
    "path": "/backup/report.pdf",
    "type": "file",
    "size": 2048576
  }

说明:
  - 复制文件时，创建新的虚拟文件记录，指向同一个对象
  - 复制目录时，递归复制所有子项
```

---

### 3. 3 空间管理服务

#### 3.3.1 功能清单

| 功能 | 接口 | 描述 | 优先级 |
|-----|------|------|-------|
| 空间概览 | `GET /space` | 获取总体空间统计 | P0 |
| 账号列表 | `GET /space/accounts` | 列出所有存储账号 | P0 |
| 账号详情 | `GET /space/accounts/{id}` | 获取单个账号详情 | P1 |
| 同步空间 | `POST /space/accounts/{id}/sync` | 同步账号空间信息 | P1 |

#### 3.3.2 接口详细定义

```yaml
# 空间概览
GET /api/v1/space

响应 (200 OK):
  {
    "total_space": 26843545600000,      # 总空间 (bytes) ~25TB
    "used_space": 5368709120000,        # 已用空间
    "available_space": 21474836480000,  # 可用空间
    "usage_percent": 20. 0,              # 使用百分比
    "account_count": 25,                # 账号数量
    "active_accounts": 24,              # 活跃账号数
    "object_count": 150000,             # 对象总数
    "bucket_count": 5                   # 桶数量
  }

# 账号列表
GET /api/v1/space/accounts

响应 (200 OK):
  {
    "accounts": [
      {
        "id": "acc_a1b2c3",
        "name": "E3-Account-01",
        "email": "dev01@xxx.onmicrosoft.com",
        "status": "active",
        "total_space": 1099511627776,
        "used_space": 214748364800,
        "available_space": 884763262976,
        "usage_percent": 19.5,
        "object_count": 6000,
        "last_sync": "2024-01-15T10:00:00Z"
      },
      ...
    ]
  }

# 同步账号空间
POST /api/v1/space/accounts/{id}/sync

描述: 从 OneDrive 同步最新的空间使用情况

响应 (200 OK):
  {
    "id": "acc_a1b2c3",
    "synced_at": "2024-01-15T10:30:00Z",
    "total_space": 1099511627776,
    "used_space": 214748364800
  }
```

---

### 3. 4 系统管理服务

#### 3.4. 1 功能清单

| 功能 | 接口 | 描述 | 优先级 |
|-----|------|------|-------|
| 健康检查 | `GET /health` | 服务健康状态 | P0 |
| 系统信息 | `GET /info` | 系统版本信息 | P1 |

```yaml
# 健康检查
GET /api/v1/health

响应 (200 OK):
  {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "components": {
      "database": "healthy",
      "cache": "healthy",
      "onedrive": "healthy"
    }
  }

# 系统信息
GET /api/v1/info

响应 (200 OK):
  {
    "name": "OneDrive Storage Middleware",
    "version": "1.0.0",
    "api_version": "v1"
  }
```

---

## 四、数据模型设计

### 4.1 ER 图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            数据模型关系图                                │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│ storage_accounts │       │     buckets      │       │     objects      │
├──────────────────┤       ├──────────────────┤       ├──────────────────┤
│ id (PK)          │       │ name (PK)        │       │ bucket (PK,FK)   │
│ name             │       │ created_at       │       │ key (PK)         │
│ email            │       │ updated_at       │       │ account_id (FK)  │◄──┐
│ client_id        │       └──────────────────┘       │ remote_id        │   │
│ client_secret    │                                  │ size             │   │
│ tenant_id        │                                  │ etag             │   │
│ refresh_token    │◄─────────────────────────────────│ mime_type        │   │
│ access_token     │                                  │ is_chunked       │   │
│ token_expires    │                                  │ created_at       │   │
│ total_space      │                                  │ updated_at       │   │
│ used_space       │                                  └──────────────────┘   │
│ status           │                                           │             │
│ priority         │                                           │ 1:N         │
│ last_sync        │                                           ▼             │
│ created_at       │                                  ┌──────────────────┐   │
│ updated_at       │                                  │  object_chunks   │   │
└──────────────────┘                                  ├──────────────────┤   │
                                                      │ id (PK)          │   │
                                                      │ bucket (FK)      │   │
                                                      │ key (FK)         │   │
┌──────────────────┐       ┌──────────────────┐       │ chunk_index      │   │
│ virtual_directories│     │  virtual_files   │       │ account_id (FK)  │───┘
├──────────────────┤       ├──────────────────┤       │ remote_id        │
│ id (PK)          │◄──────│ directory_id(FK) │       │ chunk_size       │
│ bucket (FK)      │       │ id (PK)          │       │ checksum         │
│ parent_id (FK)   │───┐   │ bucket (FK)      │       │ created_at       │
│ name             │   │   │ name             │       └──────────────────┘
│ full_path        │   │   │ full_path        │
│ created_at       │   │   │ object_key (FK)  │───────► objects(bucket,key)
└──────────────────┘   │   │ size             │
         ▲             │   │ mime_type        │
         │             │   │ created_at       │
         └─────────────┘   │ updated_at       │
          (自引用)         └──────────────────┘
```

### 4.2 表结构详细定义

```sql
-- ============================================================
-- 存储账号表
-- ============================================================
CREATE TABLE storage_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    
    -- Azure AD 应用凭证
    client_id       VARCHAR(255) NOT NULL,
    client_secret   TEXT NOT NULL,              -- 加密存储
    tenant_id       VARCHAR(255) NOT NULL,
    
    -- OAuth 令牌
    refresh_token   TEXT,                       -- 加密存储
    access_token    TEXT,
    token_expires   TIMESTAMP,
    
    -- 空间信息
    total_space     BIGINT DEFAULT 0,           -- 总空间 (bytes)
    used_space      BIGINT DEFAULT 0,           -- 已用空间 (bytes)
    
    -- 状态管理
    status          VARCHAR(50) DEFAULT 'active',
                    -- active: 正常使用
                    -- disabled: 已禁用
                    -- error: 出错
                    -- syncing: 同步中
    
    priority        INT DEFAULT 0,              -- 负载均衡权重
    last_sync       TIMESTAMP,                  -- 上次同步时间
    error_message   TEXT,                       -- 错误信息
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_accounts_status ON storage_accounts(status);
CREATE INDEX idx_accounts_priority ON storage_accounts(priority DESC);

-- ============================================================
-- 桶表
-- ============================================================
CREATE TABLE buckets (
    name            VARCHAR(63) PRIMARY KEY,    -- 桶名称
    
    -- 统计信息（定期更新）
    object_count    BIGINT DEFAULT 0,
    total_size      BIGINT DEFAULT 0,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    -- 桶名称规则：3-63字符，小写字母/数字/连字符
    CONSTRAINT bucket_name_format CHECK (
        name ~ '^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$'
    )
);

-- ============================================================
-- 对象表
-- ============================================================
CREATE TABLE objects (
    bucket          VARCHAR(63) NOT NULL REFERENCES buckets(name),
    key             VARCHAR(1024) NOT NULL,
    
    -- 存储位置
    account_id      UUID NOT NULL REFERENCES storage_accounts(id),
    remote_id       VARCHAR(255),               -- OneDrive item ID
    remote_path     TEXT,                       -- OneDrive 中的路径
    
    -- 对象属性
    size            BIGINT NOT NULL,
    etag            VARCHAR(64),                -- MD5 或 SHA256
    mime_type       VARCHAR(255),
    
    -- 分片标记
    is_chunked      BOOLEAN DEFAULT FALSE,
    chunk_count     INT DEFAULT 0,
    
    -- 自定义元数据 (JSON)
    metadata        JSONB,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (bucket, key)
);

CREATE INDEX idx_objects_account ON objects(account_id);
CREATE INDEX idx_objects_created ON objects(created_at DESC);

-- ============================================================
-- 对象分片表（大文件）
-- ============================================================
CREATE TABLE object_chunks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 所属对象
    bucket          VARCHAR(63) NOT NULL,
    key             VARCHAR(1024) NOT NULL,
    chunk_index     INT NOT NULL,               -- 分片序号 (从0开始)
    
    -- 存储位置
    account_id      UUID NOT NULL REFERENCES storage_accounts(id),
    remote_id       VARCHAR(255),
    remote_path     TEXT,
    
    -- 分片属性
    chunk_size      BIGINT NOT NULL,
    checksum        VARCHAR(64),                -- 分片校验和
    
    -- 状态
    status          VARCHAR(50) DEFAULT 'pending',
                    -- pending: 待上传
                    -- uploading: 上传中
                    -- uploaded: 已上传
                    -- error: 出错
    
    created_at      TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (bucket, key) REFERENCES objects(bucket, key) ON DELETE CASCADE,
    UNIQUE(bucket, key, chunk_index)
);

CREATE INDEX idx_chunks_object ON object_chunks(bucket, key);
CREATE INDEX idx_chunks_account ON object_chunks(account_id);

-- ============================================================
-- 虚拟目录表
-- ============================================================
CREATE TABLE virtual_directories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket          VARCHAR(63) NOT NULL REFERENCES buckets(name),
    parent_id       UUID REFERENCES virtual_directories(id) ON DELETE CASCADE,
    
    name            VARCHAR(255) NOT NULL,      -- 目录名
    full_path       TEXT NOT NULL,              -- 完整路径，如 /a/b/c/
    
    created_at      TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(bucket, parent_id, name),
    UNIQUE(bucket, full_path)
);

CREATE INDEX idx_vdir_bucket_path ON virtual_directories(bucket, full_path);
CREATE INDEX idx_vdir_parent ON virtual_directories(parent_id);

-- ============================================================
-- 虚拟文件表
-- ============================================================
CREATE TABLE virtual_files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket          VARCHAR(63) NOT NULL REFERENCES buckets(name),
    directory_id    UUID REFERENCES virtual_directories(id) ON DELETE CASCADE,
    
    name            VARCHAR(255) NOT NULL,      -- 文件名
    full_path       TEXT NOT NULL,              -- 完整路径
    
    -- 关联到实际对象
    object_key      VARCHAR(1024) NOT NULL,
    
    -- 冗余存储，加速查询
    size            BIGINT NOT NULL,
    mime_type       VARCHAR(255),
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (bucket, object_key) REFERENCES objects(bucket, key),
    UNIQUE(bucket, directory_id, name),
    UNIQUE(bucket, full_path)
);

CREATE INDEX idx_vfile_bucket_path ON virtual_files(bucket, full_path);
CREATE INDEX idx_vfile_directory ON virtual_files(directory_id);
CREATE INDEX idx_vfile_object ON virtual_files(bucket, object_key);
```

---

## 五、核心流程设计

### 5.1 对象上传流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         对象上传流程                                     │
└─────────────────────────────────────────────────────────────────────────┘

  Client                        Middleware                        OneDrive
     │                              │                                 │
     │ PUT /objects/images/abc123   │                                 │
     │ Content-Length: 50MB         │                                 │
     │ ────────────────────────────>│                                 │
     │                              │                                 │
     │                              │ (1) 验证请求参数                  │
     │                              │     检查 bucket 是否存在          │
     │                              │     检查 key 是否冲突             │
     │                              │                                 │
     │                              │ (2) 判断上传策略                  │
     │                              │     < 4MB: 小文件直传             │
     │                              │     >= 4MB: 分片上传             │
     │                              │                                 │
     │                              │ (3) 选择目标账号                  │
     │                              │     根据负载均衡策略选择           │
     │                              │                                 │
     │                              │                 ┌───────────────┐
     │                              │ (4) 小文件直传   │               │
     │                              │ ───────────────>│ PUT item      │
     │                              │                 │               │
     │                              │                 │ OR            │
     │                              │                 │               │
     │                              │ (4) 分片上传    │               │
     │                              │ ─── 创建会话 ──>│               │
     │                              │ ─── 上传分片1 ─>│               │
     │                              │ ─── 上传分片2 ─>│               │
     │                              │ ─── ...        ─>│               │
     │                              │ ─── 完成上传 ──>│               │
     │                              │                 └───────────────┘
     │                              │                                 │
     │                              │ (5) 保存元数据到数据库            │
     │                              │     objects 表                   │
     │                              │     object_chunks 表（如有）      │
     │                              │                                 │
     │  200 OK                      │                                 │
     │  { bucket, key, size, etag } │                                 │
     │ <────────────────────────────│                                 │
     │                              │                                 │
```

### 5. 2 对象下载流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         对象下载流程                                     │
└─────────────────────────────────────────────────────────────────────────┘

  Client                        Middleware                        OneDrive
     │                              │                                 │
     │ GET /objects/images/abc123   │                                 │
     │ Range: bytes=0-1048575       │                                 │
     │ ────────────────────────────>│                                 │
     │                              │                                 │
     │                              │ (1) 查询对象元数据                │
     │                              │     从 objects 表获取            │
     │                              │                                 │
     │                              │ (2) 检查是否分片                  │
     │                              │                                 │
     │                              │             ┌───────────────────┐
     │                              │             │                   │
     │                              │  非分片     │ 直接下载           │
     │                              │ ──────────>│ GET item content  │
     │                              │             │                   │
     │                              │  分片       │ 计算分片范围        │
     │                              │ ──────────>│ 并行下载相关分片    │
     │                              │             │ 拼接返回           │
     │                              │             └───────────────────┘
     │                              │                                 │
     │  200 OK / 206 Partial        │                                 │
     │  [文件内容]                   │                                 │
     │ <────────────────────────────│                                 │
     │                              │                                 │
```

### 5. 3 虚拟目录上传流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       虚拟目录上传流程                                    │
└─────────────────────────────────────────────────────────────────────────┘

  Client                           Middleware                              
     │                                 │                                   
     │ PUT /vfs/docs/reports/2024/q1. pdf                                   
     │ ───────────────────────────────>│                                   
     │                                 │                                   
     │                                 │ (1) 解析路径                      
     │                                 │     bucket: docs                  
     │                                 │     目录: /reports/2024/          
     │                                 │     文件名: q1.pdf                 
     │                                 │                                   
     │                                 │ (2) 确保目录链存在                  
     │                                 │     检查/创建 /reports/           
     │                                 │     检查/创建 /reports/2024/      
     │                                 │                                   
     │                                 │ (3) 生成对象 Key                   
     │                                 │     key: "obj_f8a3b2c1..."         
     │                                 │                                   
     │                                 │ (4) 调用对象存储服务上传            
     │                                 │     PUT /objects/docs/obj_f8a3b2c1 
     │                                 │                                   
     │                                 │ (5) 创建虚拟文件记录                
     │                                 │     virtual_files 表              
     │                                 │     path: /reports/2024/q1.pdf    
     │                                 │     object_key: obj_f8a3b2c1      
     │                                 │                                   
     │  201 Created                    │                                   
     │  { id, name, path, size, ...  }  │                                   
     │ <───────────────────────────────│                                   
     │                                 │                                   
```

### 5. 4 负载均衡策略

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         负载均衡策略                                     │
└─────────────────────────────────────────────────────────────────────────┘

策略选择优先级：

┌─────────────────────────────────────────────────────────────────────────┐
│  1. 过滤可用账号                                                         │
│     • status = 'active'                                                 │
│     • available_space >= 所需空间                                        │
│     • token 未过期或可刷新                                               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  2. 选择策略 (可配置)                                                    │
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐          │
│  │  LEAST_USED     │  │  ROUND_ROBIN    │  │    WEIGHTED     │          │
│  │  (推荐)          │  │                 │  │                 │          │
│  │                 │  │                 │  │                 │          │
│  │  选择使用率      │  │  轮询选择        │  │  按优先级权重    │          │
│  │  最低的账号      │  │  每次下一个      │  │  加权随机选择    │          │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘          │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  3. 故障转移                                                             │
│     • 如果选中账号上传失败，自动尝试下一个                                  │
│     • 记录失败次数，临时降低优先级                                         │
│     • 连续失败超过阈值，标记为 error 状态                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 六、配置设计

### 6. 1 应用配置

```yaml
# config. yaml

# 服务配置
server:
  host: "0.0.0.0"
  port: 8080
  api_prefix: "/api/v1"

# 数据库配置
database:
  host: "localhost"
  port: 5432
  name: "onedrive_storage"
  user: "postgres"
  password: "${DB_PASSWORD}"       # 从环境变量读取
  max_connections: 20

# 缓存配置
cache:
  enabled: true
  type: "redis"                    # redis 或 memory
  redis:
    host: "localhost"
    port: 6379
    password: "${REDIS_PASSWORD}"
    db: 0
  ttl:
    token: 3000                    # access_token 缓存时间（秒）
    metadata: 300                  # 元数据缓存时间（秒）

# 存储配置
storage:
  # 上传配置
  upload:
    max_file_size: 107374182400    # 最大文件大小 100GB
    chunk_size: 10485760           # 分片大小 10MB
    chunk_threshold: 4194304       # 分片阈值 4MB（超过此大小分片上传）
    parallel_chunks: 4             # 并行上传分片数
  
  # 负载均衡
  load_balance:
    strategy: "least_used"         # least_used / round_robin / weighted
    health_check_interval: 60      # 健康检查间隔（秒）
  
  # 重试配置
  retry:
    max_attempts: 3
    initial_delay: 1000            # 初始延迟（毫秒）
    max_delay: 30000               # 最大延迟（毫秒）
    multiplier: 2                  # 延迟倍数

# 令牌管理
token:
  refresh_before_expire: 300       # 过期前多少秒刷新（秒）
  refresh_check_interval: 60       # 检查刷新的间隔（秒）

# 日志配置
logging:
  level: "info"                    # debug / info / warn / error
  format: "json"                   # json / text
  output: "stdout"                 # stdout / file
  file:
    path: "/var/log/storage/app.log"
    max_size: 100                  # MB
    max_backups: 10
    max_age: 30                    # 天
```

### 6. 2 账号配置

```yaml
# accounts. yaml (或通过 API/数据库管理)

accounts:
  - name: "E3-Account-01"
    email: "dev01@contoso.onmicrosoft.com"
    client_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    client_secret: "${ACCOUNT_01_SECRET}"
    tenant_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    priority: 10
    
  - name: "E3-Account-02"
    email: "dev02@contoso.onmicrosoft.com"
    client_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    client_secret: "${ACCOUNT_02_SECRET}"
    tenant_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    priority: 10
    
  # ...  更多账号
```

---

## 七、错误处理设计

### 7.1 错误码定义

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           错误码定义                                     │
└─────────────────────────────────────────────────────────────────────────┘

HTTP 状态码  |  错误码           |  描述
─────────────┼───────────────────┼──────────────────────────────────
400          |  INVALID_REQUEST  |  请求参数无效
400          |  INVALID_BUCKET   |  桶名称格式无效
400          |  INVALID_KEY      |  对象键格式无效
400          |  INVALID_PATH     |  路径格式无效
─────────────┼───────────────────┼──────────────────────────────────
404          |  BUCKET_NOT_FOUND |  桶不存在
404          |  OBJECT_NOT_FOUND |  对象不存在
404          |  PATH_NOT_FOUND   |  路径不存在
─────────────┼───────────────────┼──────────────────────────────────
409          |  BUCKET_EXISTS    |  桶已存在
409          |  OBJECT_EXISTS    |  对象已存在
409          |  PATH_EXISTS      |  路径已存在
409          |  BUCKET_NOT_EMPTY |  桶非空，无法删除
409          |  DIR_NOT_EMPTY    |  目录非空，无法删除
─────────────┼───────────────────┼──────────────────────────────────
413          |  FILE_TOO_LARGE   |  文件超过大小限制
507          |  STORAGE_FULL     |  存储空间不足
─────────────┼───────────────────┼──────────────────────────────────
500          |  INTERNAL_ERROR   |  内部服务错误
502          |  UPSTREAM_ERROR   |  上游服务（OneDrive）错误
503          |  SERVICE_UNAVAIL  |  服务暂时不可用
```

### 7. 2 错误响应格式

```json
{
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "The specified object does not exist",
    "details": {
      "bucket": "images",
      "key": "abc123"
    },
    "request_id": "req_a1b2c3d4e5f6"
  }
}
```

---

## 八、项目结构

```
onedrive-storage-middleware/
│
├── cmd/                              # 入口
│   └── server/
│       └── main.go                   # 主程序入口
│
├── internal/                         # 内部代码（不对外暴露）
│   │
│   ├── api/                          # API 层
│   │   ├── handlers/                 # 请求处理器
│   │   │   ├── object. go             # 对象存储接口
│   │   │   ├── bucket.go             # 桶管理接口
│   │   │   ├── vfs.go                # 虚拟目录接口
│   │   │   ├── space.go              # 空间管理接口
│   │   │   └── health.go             # 健康检查接口
│   │   │
│   │   ├── middleware/               # 中间件
│   │   │   ├── auth. go               # 认证中间件
│   │   │   ├── logging.go            # 日志中间件
│   │   │   ├── recovery.go           # 错误恢复
│   │   │   └── ratelimit.go          # 限流中间件
│   │   │
│   │   ├── router. go                 # 路由定义
│   │   └── server. go                 # HTTP 服务器
│   │
│   ├── service/                      # 业务逻辑层
│   │   ├── object/                   # 对象存储服务
│   │   │   ├── service.go
│   │   │   ├── upload.go
│   │   │   └── download.go
│   │   │
│   │   ├── bucket/                   # 桶管理服务
│   │   │   └── service.go
│   │   │
│   │   ├── vfs/                      # 虚拟目录服务
│   │   │   ├── service.go
│   │   │   ├── directory.go
│   │   │   └── file.go
│   │   │
│   │   ├── space/                    # 空间管理服务
│   │   │   └── service.go
│   │   │
│   │   └── account/                  # 账号管理服务
│   │       ├── service.go
│   │       └── token.go
│   │
│   ├── repository/                   # 数据访问层
│   │   ├── object.go
│   │   ├── bucket.go
│   │   ├── chunk.go
│   │   ├── vfs.go
│   │   └── account.go
│   │
│   ├── infrastructure/               # 基础设施
│   │   ├── onedrive/                 # OneDrive 客户端
│   │   │   ├── client.go
│   │   │   ├── auth.go
│   │   │   ├── upload.go
│   │   │   └── download.go
│   │   │
│   │   ├── database/                 # 数据库
│   │   │   ├── postgres.go
│   │   │   └── migrations/
│   │   │
│   │   ├── cache/                    # 缓存
│   │   │   └── redis.go
│   │   │
│   │   └── scheduler/                # 调度器
│   │       └── token_refresh.go
│   │
│   ├── core/                         # 核心模块
│   │   ├── loadbalancer/             # 负载均衡
│   │   │   ├── balancer.go
│   │   │   └── strategies. go
│   │   │
│   │   └── chunker/                  # 分片处理
│   │       └── chunker.go
│   │
│   └── common/                       # 公共模块
│       ├── errors/                   # 错误定义
│       │   └── errors.go
│       ├── types/                    # 类型定义
│       │   └── types.go
│       └── utils/                    # 工具函数
│           └── utils.go
│
├── pkg/                              # 可对外暴露的包
│   └── client/                       # SDK 客户端（可选）
│       └── client.go
│
├── configs/                          # 配置文件
│   ├── config.yaml
│   └── config.example.yaml
│
├── migrations/                       # 数据库迁移
│   ├── 001_init. up.sql
│   └── 001_init.down.sql
│
├── scripts/                          # 脚本
│   ├── setup.sh
│   └── migrate.sh
│
├── docs/                             # 文档
│   ├── api. md
│   └── architecture.md
│
├── docker/                           # Docker 相关
│   ├── Dockerfile
│   └── docker-compose.yaml
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 九、开发里程碑

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           开发计划                                       │
└─────────────────────────────────────────────────────────────────────────┘

Phase 1: 基础框架 (Week 1-2)
├── [  ] 项目初始化、目录结构
├── [  ] 配置管理模块
├── [  ] 数据库连接和迁移
├── [  ] OneDrive 客户端封装
├── [  ] 单账号认证流程
└── [  ] 基础 HTTP 服务器

Phase 2: 对象存储核心 (Week 3-4)
├── [  ] Bucket 管理 (创建/删除/列表)
├── [  ] 对象上传 (小文件)
├── [  ] 对象下载
├── [  ] 对象删除
├── [  ] 对象列表
└── [  ] 对象元信息查询

Phase 3: 多账号与高级功能 (Week 5-6)
├── [  ] 多账号管理
├── [  ] 负载均衡策略
├── [  ] 大文件分片上传
├── [  ] 断点续传支持
├── [  ] 空间统计服务
└── [  ] 令牌自动刷新

Phase 4: 虚拟目录层 (Week 7-8)
├── [  ] 目录树管理
├── [  ] 路径解析
├── [  ] 文件上传到路径
├── [  ] 目录列表
├── [  ] 移动/重命名
└── [  ] 删除文件/目录

Phase 5: 稳定性与优化 (Week 9-10)
├── [  ] 错误处理完善
├── [  ] 重试机制
├── [  ] 缓存优化
├── [  ] 日志完善
├── [  ] 健康检查
└── [  ] 性能测试与优化

Phase 6: 文档与发布 (Week 11-12)
├── [  ] API 文档
├── [  ] 部署文档
├── [  ] Docker 镜像
├── [  ] 单元测试
├── [  ] 集成测试
└── [
