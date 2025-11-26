# OneDrive Storage Middleware

A unified storage middleware that aggregates multiple Microsoft 365 E3 OneDrive accounts into a unified storage pool, providing standardized storage service interfaces.

[中文文档](README.md)

## Quick Start

### Prerequisites

- Go 1.19 or higher
- PostgreSQL 12 or higher
- (Optional) Redis for caching

### Installation

1. Clone the repository:
```bash
git clone https://github.com/xuecangming/onedrive-storage.git
cd onedrive-storage
```

2. Install dependencies:
```bash
make deps
```

3. Set up database:
```bash
# Create database
createdb onedrive_storage

# Set database password
export DB_PASSWORD=your_password
```

4. Build and run:
```bash
make build
make run
```

The server will start on `http://localhost:8080` by default.

### Configuration

Copy the example configuration:
```bash
cp configs/config.yaml configs/config.yaml
```

Edit `configs/config.yaml` to customize your settings.

## API Documentation

### Health Check

```bash
# Check service health
curl http://localhost:8080/api/v1/health

# Get service info
curl http://localhost:8080/api/v1/info
```

### Bucket Management

```bash
# List all buckets
curl http://localhost:8080/api/v1/buckets

# Create a bucket
curl -X PUT http://localhost:8080/api/v1/buckets/my-bucket

# Delete a bucket
curl -X DELETE http://localhost:8080/api/v1/buckets/my-bucket
```

### Object Storage

```bash
# Upload an object
curl -X PUT http://localhost:8080/api/v1/objects/my-bucket/my-file.txt \
  -H "Content-Type: text/plain" \
  --data "Hello, World!"

# Download an object
curl http://localhost:8080/api/v1/objects/my-bucket/my-file.txt

# Get object metadata
curl -I http://localhost:8080/api/v1/objects/my-bucket/my-file.txt

# List objects in a bucket
curl http://localhost:8080/api/v1/objects/my-bucket

# Delete an object
curl -X DELETE http://localhost:8080/api/v1/objects/my-bucket/my-file.txt
```

## Development

### Run tests
```bash
make test
```

### Format code
```bash
make fmt
```

### Run linter
```bash
make lint
```

## Project Status

This is a work in progress. Current implementation status:

- [x] Phase 1: Basic Framework
  - [x] Project structure
  - [x] Configuration management
  - [x] Database schema and migrations
  - [x] Basic HTTP server
- [ ] Phase 2: Core Object Storage Service (In Progress)
  - [x] Bucket management (create/delete/list)
  - [x] Object upload (small files, in-memory)
  - [x] Object download
  - [x] Object deletion
  - [x] Object listing
  - [x] Object metadata queries
  - [ ] OneDrive integration
- [ ] Phase 3: Multi-Account & Advanced Features
- [ ] Phase 4: Virtual Directory Layer
- [ ] Phase 5: Stability & Optimization
- [ ] Phase 6: Documentation & Deployment

## License

MIT License
