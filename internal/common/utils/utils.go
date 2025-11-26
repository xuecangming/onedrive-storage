package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from file
func LoadConfig() (*types.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return default configuration if file not found
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config types.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&config)

	return &config, nil
}

// defaultConfig returns default configuration
func defaultConfig() *types.Config {
	return &types.Config{
		Server: types.ServerConfig{
			Host:      "0.0.0.0",
			Port:      8080,
			APIPrefix: "/api/v1",
		},
		Database: types.DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			Name:           "onedrive_storage",
			User:           "postgres",
			Password:       os.Getenv("DB_PASSWORD"),
			MaxConnections: 20,
		},
		Cache: types.CacheConfig{
			Enabled: false,
			Type:    "memory",
		},
		Storage: types.StorageConfig{
			Upload: types.UploadConfig{
				MaxFileSize:    107374182400, // 100GB
				ChunkSize:      10485760,     // 10MB
				ChunkThreshold: 4194304,      // 4MB
				ParallelChunks: 4,
			},
			LoadBalance: types.LoadBalanceConfig{
				Strategy:            "least_used",
				HealthCheckInterval: 60,
			},
			Retry: types.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1000,
				MaxDelay:     30000,
				Multiplier:   2,
			},
		},
		Token: types.TokenConfig{
			RefreshBeforeExpire:  300,
			RefreshCheckInterval: 60,
		},
		Logging: types.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(config *types.Config) {
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.Cache.Redis.Password = redisPassword
	}
}

// ValidateBucketName validates bucket name format
func ValidateBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	// Bucket name must be lowercase letters, numbers, and hyphens
	// Must start and end with alphanumeric
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`, name)
	return matched
}

// ValidateObjectKey validates object key format
func ValidateObjectKey(key string) bool {
	if len(key) < 1 || len(key) > 1024 {
		return false
	}
	// Object key should not be empty or contain only whitespace
	return strings.TrimSpace(key) != ""
}

// GenerateObjectKey generates a unique object key
func GenerateObjectKey() string {
	// Simple implementation - in production, use UUID or similar
	return fmt.Sprintf("obj_%d", os.Getpid())
}

// GenerateID generates a unique ID for entities
func GenerateID() string {
	// Simple implementation - in production, use UUID
	return fmt.Sprintf("id_%d_%d", os.Getpid(), os.Getpid())
}

