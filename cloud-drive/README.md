# OneDrive 云盘 Web 应用 (React 版本)

这是一个基于 React + TypeScript + Vite 构建的现代化网盘应用，使用 Ant Design 组件库，与后端中间件完全隔离。

## 技术栈

- **React 19** - 前端框架
- **TypeScript** - 类型安全
- **Vite** - 构建工具
- **Ant Design 6** - UI 组件库
- **React Query** - 服务端状态管理
- **Zustand** - 客户端状态管理
- **Axios** - HTTP 客户端

## 功能特性

### 核心功能
- ✅ 文件浏览（网格/列表视图切换）
- ✅ 面包屑导航
- ✅ 文件夹导航
- ✅ 文件选择（单选/多选/范围选择）
- ✅ 文件上传（拖拽 + 按钮，支持进度显示）
- ✅ 文件下载
- ✅ 文件删除（含确认对话框）
- ✅ 文件重命名
- ✅ 新建文件夹
- ✅ 文件移动/复制
- ✅ 文件预览（图片、视频、音频、文本、PDF）
- ✅ 右键上下文菜单
- ✅ 搜索过滤

### UI/UX
- ✅ 响应式布局
- ✅ 侧边栏导航
- ✅ 存储空间显示
- ✅ 上传进度面板
- ✅ 选择工具栏
- ✅ Toast 通知

## 项目结构

```
cloud-drive/
├── src/
│   ├── api/              # API 调用层（与中间件隔离）
│   │   ├── client.ts     # Axios 客户端配置
│   │   ├── vfs.ts        # 虚拟文件系统 API
│   │   ├── bucket.ts     # 存储桶 API
│   │   └── space.ts      # 空间管理 API
│   ├── components/       # React 组件
│   │   ├── layout/       # 布局组件（Header, Sidebar, Toolbar）
│   │   └── file/         # 文件相关组件
│   ├── hooks/            # 自定义 Hooks
│   ├── pages/            # 页面组件
│   ├── store/            # Zustand 状态管理
│   ├── types/            # TypeScript 类型定义
│   └── utils/            # 工具函数
├── public/               # 静态资源
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## 快速开始

### 前置条件

- Node.js 18+
- 后端中间件服务运行在 `http://localhost:8080`

### 安装依赖

```bash
cd cloud-drive
npm install
```

### 开发模式

```bash
npm run dev
```

访问 http://localhost:5173

### 生产构建

```bash
npm run build
```

构建产物在 `dist/` 目录

### 代码检查

```bash
npm run lint
```

## API 配置

默认 API 地址为 `http://localhost:8080/api/v1`，可以通过以下方式修改：

1. 在浏览器中打开应用后，点击侧边栏底部的"设置"
2. 修改 API 地址并保存

或者在代码中修改 `src/api/client.ts` 中的 `DEFAULT_API_URL`。

## 与中间件的关系

本应用是一个**完全独立**的前端应用，通过 REST API 与后端中间件通信：

```
┌─────────────────────────────────────────────────┐
│         云盘 Web 应用 (React SPA)               │
│  ┌──────────┬──────────┬──────────┬──────────┐ │
│  │ 页面组件 │ UI 组件  │ 状态管理 │ 工具函数 │ │
│  └──────────┴──────────┴──────────┴──────────┘ │
│  ┌─────────────────────────────────────────┐   │
│  │         API Client (Axios)              │   │
│  └─────────────────────────────────────────┘   │
└─────────────────┬───────────────────────────────┘
                  │ HTTP/HTTPS (REST API)
┌─────────────────▼───────────────────────────────┐
│      OneDrive Storage 中间件 (Go 后端)          │
│  /api/v1/vfs, /buckets, /accounts, etc.        │
└─────────────────────────────────────────────────┘
```

### 使用的 API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/buckets` | 列出存储桶 |
| PUT | `/api/v1/buckets/{bucket}` | 创建存储桶 |
| GET | `/api/v1/vfs/{bucket}/{path}/` | 列出目录 |
| PUT | `/api/v1/vfs/{bucket}/{path}` | 上传文件 |
| GET | `/api/v1/vfs/{bucket}/{path}` | 下载文件 |
| DELETE | `/api/v1/vfs/{bucket}/{path}` | 删除文件/目录 |
| POST | `/api/v1/vfs/{bucket}/_mkdir` | 创建目录 |
| POST | `/api/v1/vfs/{bucket}/_move` | 移动/重命名 |
| POST | `/api/v1/vfs/{bucket}/_copy` | 复制文件 |
| GET | `/api/v1/space` | 获取空间统计 |
| GET | `/api/v1/accounts` | 列出账号 |

## 浏览器兼容性

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 许可证

MIT License
