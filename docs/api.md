# API Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Endpoints

### System Health

#### GET /health
Check the health status of the service and its components.

**Response 200 OK:**
```json
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

#### GET /info
Get service information.

**Response 200 OK:**
```json
{
  "name": "OneDrive Storage Middleware",
  "version": "1.0.0",
  "api_version": "v1"
}
```

---

### Bucket Management

#### GET /buckets
List all buckets.

**Response 200 OK:**
```json
{
  "buckets": [
    {
      "name": "my-bucket",
      "object_count": 10,
      "total_size": 1024000,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### PUT /buckets/{bucket}
Create a new bucket.

**Path Parameters:**
- `bucket` (string, required): Bucket name (3-63 chars, lowercase alphanumeric and hyphens)

**Response 201 Created:**
```json
{
  "name": "my-bucket",
  "object_count": 0,
  "total_size": 0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error 409 Conflict:**
```json
{
  "error": {
    "code": "BUCKET_EXISTS",
    "message": "Bucket already exists",
    "details": {
      "bucket": "my-bucket"
    }
  }
}
```

#### DELETE /buckets/{bucket}
Delete an empty bucket.

**Path Parameters:**
- `bucket` (string, required): Bucket name

**Response 204 No Content**

**Error 409 Conflict:**
```json
{
  "error": {
    "code": "BUCKET_NOT_EMPTY",
    "message": "Bucket is not empty",
    "details": {
      "bucket": "my-bucket"
    }
  }
}
```

---

### Object Storage

#### PUT /objects/{bucket}/{key}
Upload an object to a bucket.

**Path Parameters:**
- `bucket` (string, required): Bucket name
- `key` (string, required): Object key (1-1024 chars)

**Headers:**
- `Content-Type` (string, optional): MIME type of the object (default: application/octet-stream)

**Request Body:**
Binary data

**Response 200 OK:**
```json
{
  "bucket": "my-bucket",
  "key": "my-file.txt",
  "account_id": "00000000-0000-0000-0000-000000000000",
  "remote_id": "dummy-remote",
  "remote_path": "/storage/my-bucket/my-file.txt",
  "size": 1024,
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "mime_type": "text/plain",
  "is_chunked": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### GET /objects/{bucket}/{key}
Download an object.

**Path Parameters:**
- `bucket` (string, required): Bucket name
- `key` (string, required): Object key

**Response 200 OK:**
Binary data with headers:
- `Content-Type`: MIME type
- `Content-Length`: File size
- `ETag`: Object ETag

**Error 404 Not Found:**
```json
{
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "Object not found",
    "details": {
      "bucket": "my-bucket",
      "key": "my-file.txt"
    }
  }
}
```

#### HEAD /objects/{bucket}/{key}
Get object metadata without downloading.

**Path Parameters:**
- `bucket` (string, required): Bucket name
- `key` (string, required): Object key

**Response 200 OK:**
Headers only:
- `Content-Type`: MIME type
- `Content-Length`: File size
- `ETag`: Object ETag

#### GET /objects/{bucket}
List objects in a bucket.

**Path Parameters:**
- `bucket` (string, required): Bucket name

**Query Parameters:**
- `prefix` (string, optional): Filter by key prefix
- `marker` (string, optional): Pagination marker
- `max_keys` (integer, optional): Maximum number of objects to return (1-1000, default: 1000)

**Response 200 OK:**
```json
{
  "bucket": "my-bucket",
  "prefix": "",
  "objects": [
    {
      "bucket": "my-bucket",
      "key": "file1.txt",
      "size": 1024,
      "etag": "d41d8cd98f00b204e9800998ecf8427e",
      "mime_type": "text/plain",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "is_truncated": false,
  "next_marker": ""
}
```

#### DELETE /objects/{bucket}/{key}
Delete an object.

**Path Parameters:**
- `bucket` (string, required): Bucket name
- `key` (string, required): Object key

**Response 204 No Content**

**Error 404 Not Found:**
```json
{
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "Object not found",
    "details": {
      "bucket": "my-bucket",
      "key": "my-file.txt"
    }
  }
}
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "key": "value"
    }
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request parameters |
| `INVALID_BUCKET` | 400 | Invalid bucket name format |
| `INVALID_KEY` | 400 | Invalid object key format |
| `BUCKET_NOT_FOUND` | 404 | Bucket does not exist |
| `OBJECT_NOT_FOUND` | 404 | Object does not exist |
| `BUCKET_EXISTS` | 409 | Bucket already exists |
| `OBJECT_EXISTS` | 409 | Object already exists |
| `BUCKET_NOT_EMPTY` | 409 | Cannot delete non-empty bucket |
| `FILE_TOO_LARGE` | 413 | File exceeds size limit |
| `STORAGE_FULL` | 507 | Insufficient storage space |
| `INTERNAL_ERROR` | 500 | Internal server error |

---

## Examples

### Create a bucket and upload a file

```bash
# Create bucket
curl -X PUT http://localhost:8080/api/v1/buckets/my-bucket

# Upload file
curl -X PUT http://localhost:8080/api/v1/objects/my-bucket/hello.txt \
  -H "Content-Type: text/plain" \
  --data "Hello, World!"

# Download file
curl http://localhost:8080/api/v1/objects/my-bucket/hello.txt

# List objects
curl http://localhost:8080/api/v1/objects/my-bucket

# Delete object
curl -X DELETE http://localhost:8080/api/v1/objects/my-bucket/hello.txt

# Delete bucket
curl -X DELETE http://localhost:8080/api/v1/buckets/my-bucket
```

---

## Account Management

### GET /accounts
List all storage accounts.

**Response 200 OK:**
```json
{
  "accounts": [
    {
      "id": "uuid",
      "name": "E3-Account-01",
      "email": "dev01@example.onmicrosoft.com",
      "status": "active",
      "total_space": 1099511627776,
      "used_space": 214748364800,
      "priority": 10,
      "last_sync": "2024-01-15T10:00:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ]
}
```

### POST /accounts
Create a new storage account.

**Request Body:**
```json
{
  "name": "E3-Account-01",
  "email": "dev01@example.onmicrosoft.com",
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "tenant_id": "your-tenant-id",
  "refresh_token": "your-refresh-token",
  "priority": 10
}
```

**Response 201 Created:**
```json
{
  "id": "uuid",
  "name": "E3-Account-01",
  "email": "dev01@example.onmicrosoft.com",
  "status": "active",
  "priority": 10,
  "created_at": "2024-01-15T10:00:00Z"
}
```

### GET /accounts/{id}
Get account details.

**Response 200 OK:**
```json
{
  "id": "uuid",
  "name": "E3-Account-01",
  "email": "dev01@example.onmicrosoft.com",
  "status": "active",
  "total_space": 1099511627776,
  "used_space": 214748364800,
  "priority": 10,
  "last_sync": "2024-01-15T10:00:00Z"
}
```

### PUT /accounts/{id}
Update account information.

**Request Body:**
```json
{
  "name": "E3-Account-01-Updated",
  "priority": 20
}
```

**Response 200 OK:**
```json
{
  "id": "uuid",
  "name": "E3-Account-01-Updated",
  "priority": 20,
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### DELETE /accounts/{id}
Delete an account.

**Response 204 No Content**

### POST /accounts/{id}/refresh
Refresh account's access token.

**Response 200 OK:**
```json
{
  "message": "Token refreshed successfully"
}
```

### POST /accounts/{id}/sync
Sync account space information from OneDrive.

**Response 200 OK:**
```json
{
  "id": "uuid",
  "name": "E3-Account-01",
  "total_space": 1099511627776,
  "used_space": 214748364800,
  "last_sync": "2024-01-15T10:35:00Z"
}
```

---

## Space Management

### GET /space
Get overall space statistics.

**Response 200 OK:**
```json
{
  "total_accounts": 5,
  "active_accounts": 4,
  "total_space": 5497558138880,
  "used_space": 1073741824000,
  "available_space": 4423816314880,
  "usage_percent": 19.5
}
```

### GET /space/accounts
List accounts with space information.

**Response 200 OK:**
```json
{
  "accounts": [
    {
      "id": "uuid",
      "name": "E3-Account-01",
      "email": "dev01@example.onmicrosoft.com",
      "status": "active",
      "total_space": 1099511627776,
      "used_space": 214748364800,
      "available_space": 884763262976,
      "usage_percent": 19.5,
      "priority": 10,
      "last_sync": "2024-01-15T10:00:00Z"
    }
  ]
}
```

### GET /space/accounts/{id}
Get space details for a specific account.

**Response 200 OK:**
Same as `GET /accounts/{id}`

### POST /space/accounts/{id}/sync
Sync space information for a specific account.

**Response 200 OK:**
Same as `POST /accounts/{id}/sync`

---

## Phase 3 Features

### Load Balancing
The system automatically selects the best account for storing files based on:
- **Least Used**: Selects account with lowest usage percentage (default)
- **Round Robin**: Cycles through accounts
- **Weighted**: Uses priority-based weighted random selection

### Token Management
- Automatic token refresh when token expires within 5 minutes
- Token validation before OneDrive operations
- Failed operations trigger account status update

### OneDrive Integration
- Small files (<4MB): Direct upload
- Large files (≥4MB): Chunked upload via upload session
- Automatic retry on failure
- Fallback to in-memory storage when OneDrive is disabled

