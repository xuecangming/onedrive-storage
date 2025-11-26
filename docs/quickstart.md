# Quick Start Guide

This guide will help you get the OneDrive Storage Middleware up and running quickly.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.19 or higher**: [Download Go](https://golang.org/dl/)
- **PostgreSQL 12 or higher**: [Download PostgreSQL](https://www.postgresql.org/download/)
- **Git**: [Download Git](https://git-scm.com/downloads)

Optional but recommended:
- **jq**: For pretty-printing JSON responses
- **curl**: For testing API endpoints

## Installation

### Step 1: Clone the Repository

```bash
git clone https://github.com/xuecangming/onedrive-storage.git
cd onedrive-storage
```

### Step 2: Install Dependencies

```bash
make deps
```

This will download all required Go dependencies.

### Step 3: Set Up the Database

#### Create the Database

```bash
# Using psql command line
createdb onedrive_storage

# Or using PostgreSQL client
psql -U postgres
CREATE DATABASE onedrive_storage;
\q
```

#### Set Database Password

```bash
export DB_PASSWORD=your_password
```

For production, use a secure password and store it securely.

#### Initialize the Database

The application will automatically create the necessary tables on first startup using migrations.

### Step 4: Configure the Application (Optional)

The application works with sensible defaults. If you need to customize settings:

```bash
cp configs/config.yaml configs/config.local.yaml
# Edit config.local.yaml with your settings
export CONFIG_PATH=configs/config.local.yaml
```

### Step 5: Build the Application

```bash
make build
```

This creates the binary at `bin/server`.

### Step 6: Run the Application

```bash
make run
```

Or run the binary directly:

```bash
export DB_PASSWORD=your_password
./bin/server
```

The server will start on `http://localhost:8080` by default.

## Verify the Installation

### Check Server Health

```bash
curl http://localhost:8080/api/v1/health
```

Expected response:
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

### Run the Test Suite

```bash
./scripts/test_api.sh
```

This will run a comprehensive test of all API endpoints.

## Basic Usage

### Create a Bucket

```bash
curl -X PUT http://localhost:8080/api/v1/buckets/my-first-bucket
```

### Upload a File

```bash
echo "Hello, OneDrive Storage!" > test.txt
curl -X PUT http://localhost:8080/api/v1/objects/my-first-bucket/test.txt \
  -H "Content-Type: text/plain" \
  --data-binary @test.txt
```

### List Objects

```bash
curl http://localhost:8080/api/v1/objects/my-first-bucket
```

### Download a File

```bash
curl http://localhost:8080/api/v1/objects/my-first-bucket/test.txt
```

### Delete a File

```bash
curl -X DELETE http://localhost:8080/api/v1/objects/my-first-bucket/test.txt
```

### Delete a Bucket

```bash
curl -X DELETE http://localhost:8080/api/v1/buckets/my-first-bucket
```

## Next Steps

- Read the [API Documentation](api.md) for detailed endpoint information
- See the [README](../README.md) for architecture overview
- Check the [English README](../README.en.md) for project status

## Troubleshooting

### Database Connection Issues

If you see database connection errors:

1. Ensure PostgreSQL is running:
   ```bash
   sudo service postgresql status
   ```

2. Check your database credentials:
   ```bash
   psql -U postgres -d onedrive_storage -c "SELECT 1;"
   ```

3. Verify the DB_PASSWORD environment variable is set:
   ```bash
   echo $DB_PASSWORD
   ```

### Port Already in Use

If port 8080 is already in use, you can change it in the config file:

```yaml
server:
  port: 8081  # Change to any available port
```

### Build Errors

If you encounter build errors:

1. Verify Go version:
   ```bash
   go version
   ```

2. Clean and rebuild:
   ```bash
   make clean
   make deps
   make build
   ```

## Development

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Run Linter

```bash
make lint
```

## Getting Help

If you encounter issues:

1. Check the [docs](.) directory for detailed documentation
2. Look at the server logs for error messages
3. Open an issue on GitHub

## Current Limitations (Phase 1 & 2)

- **Storage**: Currently uses in-memory storage (data is lost on restart)
- **OneDrive Integration**: Not yet implemented (Phase 3)
- **Multi-Account**: Single dummy account only (Phase 3)
- **Large Files**: Chunked upload not yet implemented (Phase 3)

These features will be added in future phases according to the development roadmap.
