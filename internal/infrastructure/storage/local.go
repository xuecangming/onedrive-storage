package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// LocalStorage provides local file system storage
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Create base directory if not exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// Store stores data to local file system
func (s *LocalStorage) Store(bucket, key string, data []byte) (string, error) {
	// Create bucket directory
	bucketPath := filepath.Join(s.basePath, bucket)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create bucket directory: %w", err)
	}

	// Write file
	filePath := filepath.Join(bucketPath, key)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// Retrieve retrieves data from local file system
func (s *LocalStorage) Retrieve(bucket, key string) ([]byte, error) {
	filePath := filepath.Join(s.basePath, bucket, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s/%s", bucket, key)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

// Delete deletes data from local file system
func (s *LocalStorage) Delete(bucket, key string) error {
	filePath := filepath.Join(s.basePath, bucket, key)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists checks if a file exists
func (s *LocalStorage) Exists(bucket, key string) bool {
	filePath := filepath.Join(s.basePath, bucket, key)
	_, err := os.Stat(filePath)
	return err == nil
}

// GetPath returns the full path for a bucket/key
func (s *LocalStorage) GetPath(bucket, key string) string {
	return filepath.Join(s.basePath, bucket, key)
}
