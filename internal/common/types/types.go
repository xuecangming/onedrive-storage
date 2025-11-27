package types

import "time"

// Config represents application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Cache    CacheConfig    `yaml:"cache"`
	Storage  StorageConfig  `yaml:"storage"`
	Token    TokenConfig    `yaml:"token"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	APIPrefix string `yaml:"api_prefix"`
	BaseURL   string `yaml:"base_url"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Name           string `yaml:"name"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	MaxConnections int    `yaml:"max_connections"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled bool        `yaml:"enabled"`
	Type    string      `yaml:"type"`
	Redis   RedisConfig `yaml:"redis"`
	TTL     TTLConfig   `yaml:"ttl"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// TTLConfig represents TTL configuration
type TTLConfig struct {
	Token    int `yaml:"token"`
	Metadata int `yaml:"metadata"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Upload      UploadConfig      `yaml:"upload"`
	LoadBalance LoadBalanceConfig `yaml:"load_balance"`
	Retry       RetryConfig       `yaml:"retry"`
}

// UploadConfig represents upload configuration
type UploadConfig struct {
	MaxFileSize    int64 `yaml:"max_file_size"`
	ChunkSize      int64 `yaml:"chunk_size"`
	ChunkThreshold int64 `yaml:"chunk_threshold"`
	ParallelChunks int   `yaml:"parallel_chunks"`
}

// LoadBalanceConfig represents load balance configuration
type LoadBalanceConfig struct {
	Strategy            string `yaml:"strategy"`
	HealthCheckInterval int    `yaml:"health_check_interval"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts  int `yaml:"max_attempts"`
	InitialDelay int `yaml:"initial_delay"`
	MaxDelay     int `yaml:"max_delay"`
	Multiplier   int `yaml:"multiplier"`
}

// TokenConfig represents token management configuration
type TokenConfig struct {
	RefreshBeforeExpire  int `yaml:"refresh_before_expire"`
	RefreshCheckInterval int `yaml:"refresh_check_interval"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string            `yaml:"level"`
	Format string            `yaml:"format"`
	Output string            `yaml:"output"`
	File   LoggingFileConfig `yaml:"file"`
}

// LoggingFileConfig represents logging file configuration
type LoggingFileConfig struct {
	Path       string `yaml:"path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// Bucket represents a storage bucket
type Bucket struct {
	Name        string    `json:"name"`
	ObjectCount int64     `json:"object_count"`
	TotalSize   int64     `json:"total_size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Object represents a storage object
type Object struct {
	Bucket     string            `json:"bucket"`
	Key        string            `json:"key"`
	AccountID  string            `json:"account_id,omitempty"`
	RemoteID   string            `json:"remote_id,omitempty"`
	RemotePath string            `json:"remote_path,omitempty"`
	Size       int64             `json:"size"`
	ETag       string            `json:"etag,omitempty"`
	MimeType   string            `json:"mime_type,omitempty"`
	IsChunked  bool              `json:"is_chunked"`
	ChunkCount int               `json:"chunk_count,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// ObjectChunk represents a chunk of a large object
type ObjectChunk struct {
	ID         string    `json:"id"`
	Bucket     string    `json:"bucket"`
	Key        string    `json:"key"`
	ChunkIndex int       `json:"chunk_index"`
	AccountID  string    `json:"account_id"`
	RemoteID   string    `json:"remote_id,omitempty"`
	RemotePath string    `json:"remote_path,omitempty"`
	ChunkSize  int64     `json:"chunk_size"`
	Checksum   string    `json:"checksum,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// StorageAccount represents a OneDrive storage account
type StorageAccount struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	ClientID     string    `json:"client_id,omitempty"`
	ClientSecret string    `json:"client_secret,omitempty"`
	TenantID     string    `json:"tenant_id,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	TokenExpires time.Time `json:"token_expires,omitempty"`
	TotalSpace   int64     `json:"total_space"`
	UsedSpace    int64     `json:"used_space"`
	Status       string    `json:"status"`
	Priority     int       `json:"priority"`
	LastSync     time.Time `json:"last_sync,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// VirtualDirectory represents a virtual directory
type VirtualDirectory struct {
	ID        string    `json:"id"`
	Bucket    string    `json:"bucket"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	FullPath  string    `json:"full_path"`
	CreatedAt time.Time `json:"created_at"`
}

// VirtualFile represents a virtual file
type VirtualFile struct {
	ID          string    `json:"id"`
	Bucket      string    `json:"bucket"`
	DirectoryID *string   `json:"directory_id,omitempty"`
	Name        string    `json:"name"`
	FullPath    string    `json:"full_path"`
	ObjectKey   string    `json:"object_key"`
	Size        int64     `json:"size"`
	MimeType    string    `json:"mime_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// VFSItem represents a virtual file system item (file or directory)
type VFSItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Type      string     `json:"type"` // "file" or "directory"
	Size      int64      `json:"size,omitempty"`
	MimeType  string     `json:"mime_type,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
