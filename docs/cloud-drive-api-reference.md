# Cloud Drive Application API Reference

This document provides a comprehensive reference for the Cloud Drive Application API. It is intended for frontend developers building the user-facing "Cloud Drive" interface (distinct from the Admin UI).

## Base URL

All API endpoints are prefixed with the API version.
**Default Base URL**: `/api/v1`

## Data Models

### VFSItem (File/Directory)
Represents a file or directory in the virtual file system.
```json
{
  "id": "string",
  "name": "string",
  "path": "string",
  "type": "file" | "directory",
  "size": 1024, // bytes
  "mime_type": "image/jpeg",
  "created_at": "2023-10-27T10:00:00Z",
  "updated_at": "2023-10-27T10:00:00Z",
  "is_starred": false
}
```

### VirtualFile
Detailed representation of a file.
```json
{
  "id": "string",
  "bucket": "string",
  "directory_id": "string", // optional
  "name": "string",
  "full_path": "string",
  "object_key": "string",
  "size": 1024,
  "mime_type": "string",
  "created_at": "2023-10-27T10:00:00Z",
  "updated_at": "2023-10-27T10:00:00Z"
}
```

### VirtualDirectory
Detailed representation of a directory.
```json
{
  "id": "string",
  "bucket": "string",
  "parent_id": "string", // optional
  "name": "string",
  "full_path": "string",
  "created_at": "2023-10-27T10:00:00Z"
}
```

### Task (Async Operation)
Represents a long-running background operation (e.g., moving a large directory).
```json
{
  "id": "string",
  "type": "copy" | "move" | "delete" | "sync",
  "status": "pending" | "running" | "completed" | "failed" | "cancelled",
  "progress": 50, // 0-100
  "result": {}, // Operation specific result
  "error": "string", // Error message if failed
  "created_at": "2023-10-27T10:00:00Z",
  "updated_at": "2023-10-27T10:05:00Z",
  "completed_at": "2023-10-27T10:05:00Z"
}
```

### StarredFile
```json
{
  "id": "string",
  "bucket": "string",
  "file_id": "string",
  "file_path": "string",
  "starred_at": "2023-10-27T10:00:00Z"
}
```

### TrashItem
```json
{
  "id": "string",
  "bucket": "string",
  "original_type": "file" | "directory",
  "original_id": "string",
  "original_path": "string",
  "original_name": "string",
  "size": 1024,
  "deleted_at": "2023-10-27T10:00:00Z",
  "expires_at": "2023-11-26T10:00:00Z"
}
```

## API Endpoints

### 1. File System Operations (VFS)

#### List Directory / Download File
**GET** `/vfs/{bucket}/{path}`

- **Parameters**:
  - `bucket`: Storage bucket name (e.g., "default")
  - `path`: Path to file or directory (e.g., "photos/vacation")
  - `type`: (Query) Set to `directory` to force directory listing.
  - `recursive`: (Query) Set to `true` for recursive listing.

- **Response (Directory)**:
  ```json
  {
    "path": "/photos/vacation",
    "items": [VFSItem, VFSItem, ...],
    "total": 2
  }
  ```

- **Response (File)**:
  - Binary file content.
  - Headers: `Content-Type`, `Content-Length`, `Last-Modified`.

#### Create Directory
**POST** `/vfs/{bucket}/_mkdir`

- **Request Body**:
  ```json
  {
    "path": "/photos/new-album"
  }
  ```
- **Response**: `VirtualDirectory` object.

#### Upload File (Simple)
**PUT** `/vfs/{bucket}/{path}`

- **Headers**: `Content-Type` (MIME type)
- **Body**: Binary file content.
- **Response**: `VirtualFile` object.

#### Move / Rename
**POST** `/vfs/{bucket}/_move`

- **Request Body**:
  ```json
  {
    "source": "/photos/old_name.jpg",
    "destination": "/photos/new_name.jpg"
  }
  ```
- **Response**:
  - If File: `VirtualFile` object (200 OK).
  - If Directory: `Task` object (202 Accepted).

#### Copy
**POST** `/vfs/{bucket}/_copy`

- **Request Body**:
  ```json
  {
    "source": "/photos/image.jpg",
    "destination": "/backup/image.jpg"
  }
  ```
- **Response**:
  - If File: `VirtualFile` object (201 Created).
  - If Directory: `Task` object (202 Accepted).

#### Delete
**DELETE** `/vfs/{bucket}/{path}`

- **Parameters**:
  - `type`: (Query) `directory` (optional, for safety).
  - `recursive`: (Query) `true` (required for non-empty directories).
- **Response**:
  - If File: 204 No Content.
  - If Directory: `Task` object (202 Accepted).

#### Get File Metadata (Head)
**HEAD** `/vfs/{bucket}/{path}`

- **Response Headers**: `Content-Type`, `Content-Length`, `ETag`, `Last-Modified`.

#### Get Thumbnail
**GET** `/vfs/{bucket}/_thumbnail`

- **Parameters**:
  - `path`: (Query) Path to the image file.
  - `size`: (Query) `small`, `medium` (default), `large`.
- **Response**: Binary image data (e.g., JPEG/PNG).

---

### 2. Multipart Upload (Large Files)

#### Initiate Upload
**POST** `/vfs/{bucket}/_upload/init`

- **Request Body**:
  ```json
  {
    "path": "/videos/movie.mp4",
    "mime_type": "video/mp4"
  }
  ```
- **Response**:
  ```json
  {
    "upload_id": "unique_upload_id_123"
  }
  ```

#### Upload Part
**PUT** `/vfs/{bucket}/_upload/{uploadId}`

- **Parameters**:
  - `partNumber`: (Query) 1-based index (1, 2, 3...).
- **Body**: Binary chunk content.
- **Response**: 200 OK.

#### Complete Upload
**POST** `/vfs/{bucket}/_upload/{uploadId}/complete`

- **Request Body**:
  ```json
  {
    "path": "/videos/movie.mp4",
    "total_size": 104857600,
    "mime_type": "video/mp4"
  }
  ```
- **Response**: `VirtualFile` object.

#### List Parts
**GET** `/vfs/{bucket}/_upload/{uploadId}`

- **Response**:
  ```json
  {
    "upload_id": "...",
    "parts": [
      { "part_number": 1, "size": 5242880, "etag": "..." }
    ]
  }
  ```

#### Abort Upload
**DELETE** `/vfs/{bucket}/_upload/{uploadId}`

- **Response**: 204 No Content.

---

### 3. Enhanced Features (Drive)

#### Search
**GET** `/vfs/{bucket}/_search`

- **Parameters**:
  - `q`: (Query) Search query string.
  - `limit`: (Query) Max results (default 50).
  - `type`: (Query) Filter by type (e.g., `image`, `video`, `document`).
- **Response**:
  ```json
  {
    "query": "vacation",
    "results": [SearchResult, SearchResult, ...]
  }
  ```

#### Recent Files
**GET** `/vfs/{bucket}/_files/recent`

- **Parameters**:
  - `limit`: (Query) Default 20.
- **Response**:
  ```json
  {
    "items": [RecentFile, RecentFile, ...],
    "total": 10
  }
  ```

#### Starred Files
- **List**: **GET** `/vfs/{bucket}/_starred`
  - Response: `{ "items": [StarredFile], "total": N }`
- **Star**: **POST** `/vfs/{bucket}/_starred`
  - Body: `{ "file_id": "...", "file_path": "..." }`
- **Unstar**: **DELETE** `/vfs/{bucket}/_starred/{file_id}`

#### Trash / Recycle Bin
- **List**: **GET** `/vfs/{bucket}/_trash`
  - Response: `{ "items": [TrashItem], "total": N }`
- **Restore**: **POST** `/vfs/{bucket}/_trash/{trash_id}/restore`
- **Delete Permanently**: **DELETE** `/vfs/{bucket}/_trash/{trash_id}`
- **Empty Trash**: **DELETE** `/vfs/{bucket}/_trash`

---

### 4. Async Tasks

#### List Tasks
**GET** `/tasks`

- **Response**: `[Task, Task, ...]`

#### Get Task Status
**GET** `/tasks/{id}`

- **Response**: `Task` object.
- **Usage**: Poll this endpoint for operations that return 202 Accepted (e.g., recursive delete, move directory).

