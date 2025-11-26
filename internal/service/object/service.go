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
	"github.com/xuecangming/onedrive-storage/internal/core/loadbalancer"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/onedrive"
	"github.com/xuecangming/onedrive-storage/internal/repository"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

// Service provides object storage operations
type Service struct {
	objectRepo     *repository.ObjectRepository
	bucketRepo     *repository.BucketRepository
	accountService *account.Service
	balancer       *loadbalancer.Balancer
	useOneDrive    bool              // Flag to enable/disable OneDrive
	storage        map[string][]byte // In-memory storage fallback
}

// NewService creates a new object service
func NewService(objectRepo *repository.ObjectRepository, bucketRepo *repository.BucketRepository) *Service {
	return &Service{
		objectRepo:  objectRepo,
		bucketRepo:  bucketRepo,
		useOneDrive: false, // Keep using in-memory by default for backward compatibility
		storage:     make(map[string][]byte),
		balancer:    loadbalancer.NewBalancer(loadbalancer.StrategyLeastUsed),
	}
}

// NewServiceWithOneDrive creates a new object service with OneDrive integration
func NewServiceWithOneDrive(objectRepo *repository.ObjectRepository, bucketRepo *repository.BucketRepository, accountService *account.Service) *Service {
	return &Service{
		objectRepo:     objectRepo,
		bucketRepo:     bucketRepo,
		accountService: accountService,
		balancer:       loadbalancer.NewBalancer(loadbalancer.StrategyLeastUsed),
		useOneDrive:    true,
		storage:        make(map[string][]byte), // Fallback
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

	var accountID, remoteID, remotePath string

	// Upload to OneDrive if enabled
	if s.useOneDrive && s.accountService != nil {
		// Get active accounts
		accounts, err := s.accountService.GetActiveAccounts(ctx)
		if err != nil {
			return nil, err
		}

		// Select account using load balancer
		account, err := s.balancer.SelectAccount(ctx, accounts, int64(len(data)))
		if err != nil {
			return nil, errors.StorageFull()
		}

		// Ensure token is valid
		if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
			return nil, err
		}

		// Get fresh account with updated token
		account, err = s.accountService.Get(ctx, account.ID)
		if err != nil {
			return nil, err
		}

		// Create OneDrive client
		client := onedrive.NewClient(account.AccessToken)

		// Upload file to OneDrive
		path := fmt.Sprintf("%s/%s", bucket, key)
		item, err := client.UploadSmallFile(ctx, path, data)
		if err != nil {
			return nil, errors.UpstreamError(err.Error())
		}

		accountID = account.ID
		remoteID = item.ID
		remotePath = path
	} else {
		// Store data in memory (Phase 1/2 - fallback)
		storageKey := fmt.Sprintf("%s/%s", bucket, key)
		s.storage[storageKey] = data

		// Use dummy account for in-memory storage
		accountID = "00000000-0000-0000-0000-000000000000"
		remoteID = "dummy-remote"
		remotePath = fmt.Sprintf("/storage/%s/%s", bucket, key)
	}

	// Create object metadata
	obj := &types.Object{
		Bucket:     bucket,
		Key:        key,
		AccountID:  accountID,
		RemoteID:   remoteID,
		RemotePath: remotePath,
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

	var data []byte

	// Download from OneDrive if enabled and not using dummy account
	if s.useOneDrive && s.accountService != nil && obj.AccountID != "00000000-0000-0000-0000-000000000000" {
		// Get account
		account, err := s.accountService.Get(ctx, obj.AccountID)
		if err != nil {
			return nil, nil, err
		}

		// Ensure token is valid
		if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
			return nil, nil, err
		}

		// Get fresh account with updated token
		account, err = s.accountService.Get(ctx, account.ID)
		if err != nil {
			return nil, nil, err
		}

		// Create OneDrive client
		client := onedrive.NewClient(account.AccessToken)

		// Download file from OneDrive
		data, err = client.DownloadFile(ctx, obj.RemoteID)
		if err != nil {
			return nil, nil, errors.UpstreamError(err.Error())
		}
	} else {
		// Retrieve data from in-memory storage
		storageKey := fmt.Sprintf("%s/%s", bucket, key)
		var exists bool
		data, exists = s.storage[storageKey]
		if !exists {
			return nil, nil, errors.ObjectNotFound(bucket, key)
		}
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
	// Get object metadata first to get remote ID
	obj, err := s.objectRepo.Get(ctx, bucket, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.ObjectNotFound(bucket, key)
		}
		return errors.InternalError(err.Error())
	}

	// Delete from OneDrive if enabled and not using dummy account
	if s.useOneDrive && s.accountService != nil && obj.AccountID != "00000000-0000-0000-0000-000000000000" {
		// Get account
		account, err := s.accountService.Get(ctx, obj.AccountID)
		if err != nil {
			return err
		}

		// Ensure token is valid
		if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
			return err
		}

		// Get fresh account with updated token
		account, err = s.accountService.Get(ctx, account.ID)
		if err != nil {
			return err
		}

		// Create OneDrive client
		client := onedrive.NewClient(account.AccessToken)

		// Delete file from OneDrive
		if err := client.DeleteFile(ctx, obj.RemoteID); err != nil {
			return errors.UpstreamError(err.Error())
		}
	} else {
		// Delete from in-memory storage
		storageKey := fmt.Sprintf("%s/%s", bucket, key)
		delete(s.storage, storageKey)
	}

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
