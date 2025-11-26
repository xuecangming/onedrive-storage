package bucket

import (
	"context"
	"database/sql"

	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/repository"
)

// Service provides bucket management operations
type Service struct {
	repo *repository.BucketRepository
}

// NewService creates a new bucket service
func NewService(repo *repository.BucketRepository) *Service {
	return &Service{repo: repo}
}

// List returns all buckets
func (s *Service) List(ctx context.Context) ([]*types.Bucket, error) {
	return s.repo.List(ctx)
}

// Create creates a new bucket
func (s *Service) Create(ctx context.Context, name string) (*types.Bucket, error) {
	// Validate bucket name
	if !utils.ValidateBucketName(name) {
		return nil, errors.InvalidBucket(name)
	}

	// Check if bucket already exists
	exists, err := s.repo.Exists(ctx, name)
	if err != nil {
		return nil, errors.InternalError(err.Error())
	}
	if exists {
		return nil, errors.BucketExists(name)
	}

	// Create bucket
	bucket, err := s.repo.Create(ctx, name)
	if err != nil {
		return nil, errors.InternalError(err.Error())
	}

	return bucket, nil
}

// Delete deletes a bucket
func (s *Service) Delete(ctx context.Context, name string) error {
	// Check if bucket exists
	exists, err := s.repo.Exists(ctx, name)
	if err != nil {
		return errors.InternalError(err.Error())
	}
	if !exists {
		return errors.BucketNotFound(name)
	}

	// Check if bucket is empty
	isEmpty, err := s.repo.IsEmpty(ctx, name)
	if err != nil {
		return errors.InternalError(err.Error())
	}
	if !isEmpty {
		return errors.BucketNotEmpty(name)
	}

	// Delete bucket
	if err := s.repo.Delete(ctx, name); err != nil {
		if err == sql.ErrNoRows {
			return errors.BucketNotFound(name)
		}
		return errors.InternalError(err.Error())
	}

	return nil
}

// Get retrieves a bucket by name
func (s *Service) Get(ctx context.Context, name string) (*types.Bucket, error) {
	bucket, err := s.repo.Get(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.BucketNotFound(name)
		}
		return nil, errors.InternalError(err.Error())
	}

	return bucket, nil
}

// Exists checks if a bucket exists
func (s *Service) Exists(ctx context.Context, name string) (bool, error) {
	return s.repo.Exists(ctx, name)
}
