# 网盘应用前端开发对接文档 (Cloud Drive App Integration Guide)

本文档专为**用户端网盘应用 (User Cloud Drive App)** 的前端开发提供接口规范和交互指南。与后台管理面板 (Admin UI) 不同，网盘应用侧重于文件浏览体验、多媒体展示、文件操作以及个性化功能（如收藏、回收站）。

## 1. 基础概念

*   **Base URL**: `/api/v1`
*   **Bucket (存储桶)**: 网盘中的顶层逻辑隔离单元。通常一个用户对应一个 Bucket，或者一个组织对应一个 Bucket。前端在调用 VFS 接口时需指定 `{bucket}`。
*   **VFS (虚拟文件系统)**: 后端通过 VFS 层屏蔽了底层对象存储（OneDrive/Local）的差异，提供类似文件系统的层级结构。

## 2. 核心功能模块

### 2.1 文件浏览 (File Explorer)

网盘的核心视图，支持列表模式和网格模式。

*   **获取文件列表**:
    *   `GET /vfs/{bucket}/{path}?type=directory`
    *   **参数**: `path` (当前路径，根目录为 `/`)
    *   **响应**: 返回该目录下的子文件夹和文件列表。
*   **面包屑导航**:
    *   前端需根据当前 `path` 解析生成面包屑（e.g., `/photos/2023` -> `Home > photos > 2023`）。

### 2.2 文件操作 (File Operations)

#### 上传 (Upload)
需实现智能分流上传策略：
1.  **小文件 (<10MB)**: `PUT /vfs/{bucket}/{path}` (直接二进制流)
2.  **大文件 (>10MB)**: 分片上传流程 (Init -> Upload Parts -> Complete)。
    *   *详细流程请参考通用开发指南中的“大文件分片上传”章节。*

#### 下载 (Download)
*   `GET /vfs/{bucket}/{path}`
*   **预览**: 对于图片、PDF 等浏览器支持的格式，可直接在 iframe 或新标签页打开此 URL 进行预览。

#### 文件管理
*   **新建文件夹**: `POST /vfs/{bucket}/_mkdir`
*   **重命名/移动**: `POST /vfs/{bucket}/_move` (支持跨目录移动)
*   **复制**: `POST /vfs/{bucket}/_copy`
*   **删除**: `DELETE /vfs/{bucket}/{path}` (目录需加 `recursive=true`)

### 2.3 异步任务反馈 (Async Tasks)

对于耗时操作（如移动包含大量文件的文件夹），后端会返回 **异步任务**。

*   **场景**: 移动/复制/删除大文件夹。
*   **交互**:
    1.  调用接口收到 `202 Accepted`。
    2.  前端显示“正在处理中...”通知或进度条。
    3.  轮询 `GET /tasks/{id}` 直到状态变为 `completed`。
    4.  操作完成后自动刷新当前文件列表。

## 3. 增强体验功能 (Enhanced Features)

这些功能是网盘应用区别于普通文件管理器的关键。

### 3.1 缩略图与网格视图 (Thumbnails)
用于图片墙或网格视图模式。

*   **接口**: `GET /vfs/{bucket}/_thumbnail?path={path}&size={small|medium|large}`
*   **前端实现**:
    *   使用 `IntersectionObserver` 实现懒加载。
    *   加载失败时回退到默认图标。

### 3.2 全局搜索 (Search)
*   **接口**: `GET /vfs/{bucket}/_search?q={keyword}`
*   **功能**: 搜索文件名，支持模糊匹配。

### 3.3 最近文件 (Recent Files)
用于“首页”或“最近使用”视图。

*   **接口**: `GET /vfs/{bucket}/_files/recent`
*   **展示**: 按 `updated_at` 倒序排列的最近修改过的文件列表。

### 3.4 收藏夹 (Starred/Favorites)
用户标记重要文件。

*   **列表**: `GET /vfs/{bucket}/_starred`
*   **收藏**: `POST /vfs/{bucket}/_starred` (Body: `{ "path": "..." }`)
*   **取消收藏**: `DELETE /vfs/{bucket}/_starred/{file_id}` (注意这里通常用 file_id 或 path，具体视后端实现，建议优先使用 path 如果接口支持，或者从列表获取 id)。

### 3.5 回收站 (Trash Bin)
防止误删，提供后悔药。

*   **列表**: `GET /vfs/{bucket}/_trash`
*   **还原**: `POST /vfs/{bucket}/_trash/{trash_id}/restore`
*   **彻底删除**: `DELETE /vfs/{bucket}/_trash/{trash_id}`
*   **清空回收站**: `DELETE /vfs/{bucket}/_trash`

## 4. 数据模型 (TypeScript Interfaces)

### FileSystemItem
```typescript
export interface VFSItem {
  id: string;
  name: string;
  path: string;
  type: 'file' | 'directory';
  size: number;
  mime_type: string;
  is_starred?: boolean; // 如果后端支持在列表中返回收藏状态
  thumbnail_url?: string; // 前端构造: /api/v1/vfs/{bucket}/_thumbnail?path=...
  updated_at: string;
}
```

### TrashItem
```typescript
export interface TrashItem {
  id: string;
  original_path: string;
  name: string;
  deleted_at: string;
  size: number;
  type: 'file' | 'directory';
}
```

## 5. 典型页面路由规划

建议前端应用采用以下路由结构：

| 路径 | 页面/视图 | 描述 |
| :--- | :--- | :--- |
| `/drive/my-files/{path...}` | **我的文件** | 核心文件浏览页，根据 path 动态加载内容 |
| `/drive/recent` | **最近** | 展示最近修改的文件 |
| `/drive/starred` | **收藏** | 展示收藏的文件列表 |
| `/drive/trash` | **回收站** | 展示已删除项目 |
| `/drive/search?q=...` | **搜索结果** | 展示搜索结果 |

## 6. 错误处理建议

*   **404 Not Found**: 文件不存在或已在其他端被删除 -> 刷新列表。
*   **409 Conflict**: 目标文件已存在 -> 提示用户“覆盖”或“保留两者”。
*   **401 Unauthorized**: 登录失效 -> 跳转登录页。
*   **202 Accepted**: 异步任务开始 -> 启动轮询机制。
