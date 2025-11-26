package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(config types.DatabaseConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(config.MaxConnections)
	db.SetMaxIdleConns(config.MaxConnections / 2)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations runs database migrations
func RunMigrations(db *sql.DB) error {
	migrations := []string{
		createStorageAccountsTable,
		createBucketsTable,
		createObjectsTable,
		createObjectChunksTable,
		createVirtualDirectoriesTable,
		createVirtualFilesTable,
		insertDummyAccount,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

const createStorageAccountsTable = `
CREATE TABLE IF NOT EXISTS storage_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    
    client_id       VARCHAR(255) NOT NULL,
    client_secret   TEXT NOT NULL,
    tenant_id       VARCHAR(255) NOT NULL,
    
    refresh_token   TEXT,
    access_token    TEXT,
    token_expires   TIMESTAMP,
    
    total_space     BIGINT DEFAULT 0,
    used_space      BIGINT DEFAULT 0,
    
    status          VARCHAR(50) DEFAULT 'active',
    priority        INT DEFAULT 0,
    last_sync       TIMESTAMP,
    error_message   TEXT,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_status ON storage_accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_priority ON storage_accounts(priority DESC);
`

const createBucketsTable = `
CREATE TABLE IF NOT EXISTS buckets (
    name            VARCHAR(63) PRIMARY KEY,
    
    object_count    BIGINT DEFAULT 0,
    total_size      BIGINT DEFAULT 0,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT bucket_name_format CHECK (
        name ~ '^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$'
    )
);
`

const createObjectsTable = `
CREATE TABLE IF NOT EXISTS objects (
    bucket          VARCHAR(63) NOT NULL,
    key             VARCHAR(1024) NOT NULL,
    
    account_id      UUID NOT NULL,
    remote_id       VARCHAR(255),
    remote_path     TEXT,
    
    size            BIGINT NOT NULL,
    etag            VARCHAR(64),
    mime_type       VARCHAR(255),
    
    is_chunked      BOOLEAN DEFAULT FALSE,
    chunk_count     INT DEFAULT 0,
    
    metadata        JSONB,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (bucket, key),
    FOREIGN KEY (bucket) REFERENCES buckets(name),
    FOREIGN KEY (account_id) REFERENCES storage_accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_objects_account ON objects(account_id);
CREATE INDEX IF NOT EXISTS idx_objects_created ON objects(created_at DESC);
`

const createObjectChunksTable = `
CREATE TABLE IF NOT EXISTS object_chunks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    bucket          VARCHAR(63) NOT NULL,
    key             VARCHAR(1024) NOT NULL,
    chunk_index     INT NOT NULL,
    
    account_id      UUID NOT NULL,
    remote_id       VARCHAR(255),
    remote_path     TEXT,
    
    chunk_size      BIGINT NOT NULL,
    checksum        VARCHAR(64),
    
    status          VARCHAR(50) DEFAULT 'pending',
    
    created_at      TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (bucket, key) REFERENCES objects(bucket, key) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES storage_accounts(id),
    UNIQUE(bucket, key, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_chunks_object ON object_chunks(bucket, key);
CREATE INDEX IF NOT EXISTS idx_chunks_account ON object_chunks(account_id);
`

const createVirtualDirectoriesTable = `
CREATE TABLE IF NOT EXISTS virtual_directories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket          VARCHAR(63) NOT NULL,
    parent_id       UUID,
    
    name            VARCHAR(255) NOT NULL,
    full_path       TEXT NOT NULL,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (bucket) REFERENCES buckets(name),
    FOREIGN KEY (parent_id) REFERENCES virtual_directories(id) ON DELETE CASCADE,
    UNIQUE(bucket, parent_id, name),
    UNIQUE(bucket, full_path)
);

CREATE INDEX IF NOT EXISTS idx_vdir_bucket_path ON virtual_directories(bucket, full_path);
CREATE INDEX IF NOT EXISTS idx_vdir_parent ON virtual_directories(parent_id);
`

const createVirtualFilesTable = `
CREATE TABLE IF NOT EXISTS virtual_files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket          VARCHAR(63) NOT NULL,
    directory_id    UUID,
    
    name            VARCHAR(255) NOT NULL,
    full_path       TEXT NOT NULL,
    
    object_key      VARCHAR(1024) NOT NULL,
    
    size            BIGINT NOT NULL,
    mime_type       VARCHAR(255),
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (bucket) REFERENCES buckets(name),
    FOREIGN KEY (directory_id) REFERENCES virtual_directories(id) ON DELETE CASCADE,
    FOREIGN KEY (bucket, object_key) REFERENCES objects(bucket, key),
    UNIQUE(bucket, directory_id, name),
    UNIQUE(bucket, full_path)
);

CREATE INDEX IF NOT EXISTS idx_vfile_bucket_path ON virtual_files(bucket, full_path);
CREATE INDEX IF NOT EXISTS idx_vfile_directory ON virtual_files(directory_id);
CREATE INDEX IF NOT EXISTS idx_vfile_object ON virtual_files(bucket, object_key);
`

const insertDummyAccount = `
INSERT INTO storage_accounts (
    id, name, email, client_id, client_secret, tenant_id, status
)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'In-Memory Storage',
    'dummy@localhost',
    'dummy-client',
    'dummy-secret',
    'dummy-tenant',
    'active'
)
ON CONFLICT (id) DO NOTHING;
`
