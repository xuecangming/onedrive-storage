# 前端对接开发指南 (Frontend Integration Guide)

本文档详细描述了前端应用（Admin UI）与 OneDrive Storage 后端服务对接所需的 API 规范、交互流程及注意事项。

## 1. 基础配置

*   **Base URL**: `/api/v1`
*   **Content-Type**: 默认使用 `application/json`
*   **日期格式**: ISO 8601 (e.g., `2023-10-27T10:00:00Z`)

## 2. 核心交互流程

### 2.1 文件浏览 (VFS)

*   **获取文件列表**:
    *   `GET /vfs/{bucket}/{path}?type=directory`
    *   **注意**: `path` 必须以 `/` 开头。根目录为 `/`。
    *   **响应**: 包含 `items` 数组，区分 `type: "file"` 和 `type: "directory"`。

### 2.2 文件上传 (核心复杂点)

前端需根据文件大小自动选择上传策略。

**策略 A: 小文件上传 (<= 10MB)**
*   直接调用: `PUT /vfs/{bucket}/{path}`
*   Body: 文件二进制流
*   Header: `Content-Type: {mime_type}`

**策略 B: 大文件分片上传 (> 10MB)**
1.  **初始化**:
    *   `POST /vfs/{bucket}/_upload/init`
    *   Body: `{ "path": "/full/path/to/file.ext", "mime_type": "..." }`
    *   Resp: `{ "upload_id": "uuid..." }`
2.  **并发上传分片**:
    *   将文件切分为 **10MB** (10 * 1024 * 1024 bytes) 的块。
    *   `PUT /vfs/{bucket}/_upload/{upload_id}?partNumber={i}` (i 从 0 或 1 开始，后端目前实现似乎是索引，建议从 0 开始测试，或者查看 `service.go` 逻辑。*注：后端 `uploadOneChunk` 使用 `index`，通常是 0-based*)。
    *   **建议并发数**: 3-5。
    *   **重试**: 单个分片失败应重试 3 次。
3.  **完成上传**:
    *   `POST /vfs/{bucket}/_upload/{upload_id}/complete`
    *   Body: `{ "path": "...", "total_size": 123456, "mime_type": "..." }`
    *   **注意**: 后端会校验 `total_size` 与所有分片大小之和是否一致。

### 2.3 异步任务 (Async Tasks)

对于耗时的目录操作（复制、移动、递归删除），后端采用异步处理模式。

*   **触发操作**:
    *   复制: `POST /vfs/{bucket}/_copy`
    *   移动: `POST /vfs/{bucket}/_move`
    *   删除目录: `DELETE /vfs/{bucket}/{path}?recursive=true`
*   **响应处理**:
    *   如果返回 **202 Accepted**，Body 中会包含任务信息：`{ "id": "task-uuid", "status": "pending", ... }`。
    *   如果返回 **200 OK** 或 **204 No Content**，表示操作已同步完成（通常是单文件操作）。
*   **轮询机制**:
    *   前端需实现 `TaskPoller`。
    *   接口: `GET /tasks/{id}`
    *   频率: 建议 1-2 秒一次。
    *   **终止条件**:
        *   `status == "completed"`: 操作成功，刷新文件列表。
        *   `status == "failed"`: 操作失败，显示 `error` 字段中的信息。

### 2.4 缩略图 (Thumbnails)

*   接口: `GET /vfs/{bucket}/_thumbnail?path={path}&size={small|medium|large}`
*   **注意事项**:
    *   **懒加载**: 仅当图片进入视口时加载。
    *   **错误处理**: 必须监听 `img` 标签的 `onError` 事件。如果返回 404 或 500，应替换为默认的文件图标（后端可能因为文件不是图片或 OneDrive 生成失败而返回错误）。
    *   **缓存**: 浏览器会自动缓存，无需额外处理。

## 3. 数据结构参考

### VFS Item
```typescript
interface VFSItem {
  id: string;
  name: string;      // 显示名称 (e.g., "photo.jpg")
  path: string;      // 完整路径 (e.g., "/photos/photo.jpg")
  type: 'file' | 'directory';
  size: number;      // 字节数
  mime_type: string;
  updated_at: string;
}
```

### Task
```typescript
interface Task {
  id: string;
  type: 'copy' | 'move' | 'delete';
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;  // 0-100 (目前后端可能仅支持 0 和 100)
  result?: any;
  error?: string;
  created_at: string;
}
```

### Error Response
```json
{
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "file not found: /path/to/file"
  }
}
```

## 4. 开发注意事项 (Gotchas)

1.  **路径规范**: 所有 API 中的 `path` 参数建议始终以 `/` 开头。
2.  **URL 编码**: 在 URL 中传递路径参数（如 `GET` 请求）时，务必使用 `encodeURIComponent`，因为路径中可能包含空格或特殊字符。
3.  **删除保护**: 删除非空目录时，必须传递 `recursive=true` 参数，否则后端会返回 409 Conflict。
4.  **分片大小**: 后端默认分片大小为 **10MB**。前端切片时请严格遵守此大小（最后一个分片除外），否则可能导致合并失败或云端存储异常。
5.  **任务超时**: 虽然是异步任务，但轮询也应设置超时机制（如 5 分钟），避免因后端任务卡死导致前端无限轮询。

## 5. 调试建议

*   使用 `scripts/verify_backend.sh` 脚本可以快速在本地产生测试数据（创建 Bucket、上传文件等），方便前端开发时有数据可展示。
*   查看 `server.log` 可以获取详细的后端错误堆栈。
