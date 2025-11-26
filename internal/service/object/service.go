package object

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/repository"
)

// Service provides object storage operations
type Service struct {
	objectRepo *repository.ObjectRepository
	bucketRepo *repository.BucketRepository
	storage    map[string][]byte // In-memory storage for now (Phase 1)
}

// NewService creates a new object service
func NewService(objectRepo *repository.ObjectRepository, bucketRepo *repository.BucketRepository) *Service {
	return &Service{
		objectRepo: objectRepo,
		bucketRepo: bucketRepo,
		storage:    make(map[string][]byte),
	}
}

// Upload uploads an object
func (s *Service) Upload(ctx context.Context, bucket, key string, data []byte, mimeType string) (*types.Object, error) {
	// Validate bucket name
	if !utils.ValidateBucketName(bucket) {
		return nil, errors.InvalidBucket(bucket)
	}

	// Validate object key
	if !utils.ValidateObjectKey(key) {
		return nil, errors.InvalidKey(key)
	}

	// Check if bucket exists
	exists, err := s.bucketRepo.Exists(ctx, bucket)
	if err != nil {
		return nil, errors.InternalError(err.Error())
	}
	if !exists {
		return nil, errors.BucketNotFound(bucket)
	}

	// Calculate ETag (MD5 hash)
	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	// Store data in memory (Phase 1 - simplified)
	storageKey := fmt.Sprintf("%s/%s", bucket, key)
	s.storage[storageKey] = data

	// Create object metadata
	// Phase 1: Using a fixed UUID for the dummy account
	obj := &types.Object{
		Bucket:     bucket,
		Key:        key,
		AccountID:  "00000000-0000-0000-0000-000000000000", // Phase 1: placeholder UUID
		RemoteID:   "dummy-remote",
		RemotePath: fmt.Sprintf("/storage/%s/%s", bucket, key),
		Size:       int64(len(data)),
		ETag:       etag,
		MimeType:   mimeType,
		IsChunked:  false,
		ChunkCount: 0,
		Metadata:   make(map[string]string),
	}

	// Save to database
	if err := s.objectRepo.Create(ctx, obj); err != nil {
		return nil, errors.InternalError(err.Error())
	}

	// Update bucket stats
	s.objectRepo.UpdateBucketStats(ctx, bucket)

	return obj, nil
}

// Download downloads an object
func (s *Service) Download(ctx context.Context, bucket, key string) (*types.Object, []byte, error) {
	// Get object metadata
	obj, err := s.objectRepo.Get(ctx, bucket, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.ObjectNotFound(bucket, key)
		}
		return nil, nil, errors.InternalError(err.Error())
	}

	// Retrieve data from storage
	storageKey := fmt.Sprintf("%s/%s", bucket, key)
	data, exists := s.storage[storageKey]
	if !exists {
		return nil, nil, errors.ObjectNotFound(bucket, key)
	}

	return obj, data, nil
}

// GetMetadata retrieves object metadata
func (s *Service) GetMetadata(ctx context.Context, bucket, key string) (*types.Object, error) {
	obj, err := s.objectRepo.Get(ctx, bucket, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ObjectNotFound(bucket, key)
		}
		return nil, errors.InternalError(err.Error())
	}

	return obj, nil
}

// Delete deletes an object
func (s *Service) Delete(ctx context.Context, bucket, key string) error {
	// Check if object exists
	exists, err := s.objectRepo.Exists(ctx, bucket, key)
	if err != nil {
		return errors.InternalError(err.Error())
	}
	if !exists {
		return errors.ObjectNotFound(bucket, key)
	}

	// Delete from storage
	storageKey := fmt.Sprintf("%s/%s", bucket, key)
	delete(s.storage, storageKey)

	// Delete from database
	if err := s.objectRepo.Delete(ctx, bucket, key); err != nil {
		if err == sql.ErrNoRows {
			return errors.ObjectNotFound(bucket, key)
		}
		return errors.InternalError(err.Error())
	}

	// Update bucket stats
	s.objectRepo.UpdateBucketStats(ctx, bucket)

	return nil
}

// List lists objects in a bucket
func (s *Service) List(ctx context.Context, bucket, prefix, marker string, maxKeys int) ([]*types.Object, string, bool, error) {
	// Check if bucket exists
	exists, err := s.bucketRepo.Exists(ctx, bucket)
	if err != nil {
		return nil, "", false, errors.InternalError(err.Error())
	}
	if !exists {
		return nil, "", false, errors.BucketNotFound(bucket)
	}

	// Get objects
	objects, err := s.objectRepo.List(ctx, bucket, prefix, marker, maxKeys+1)
	if err != nil {
		return nil, "", false, errors.InternalError(err.Error())
	}

	// Check if truncated
	isTruncated := len(objects) > maxKeys
	nextMarker := ""
	if isTruncated {
		objects = objects[:maxKeys]
		nextMarker = objects[maxKeys-1].Key
	}

	return objects, nextMarker, isTruncated, nil
}
