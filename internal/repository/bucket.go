package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// BucketRepository handles bucket data access
type BucketRepository struct {
	db *sql.DB
}

// NewBucketRepository creates a new bucket repository
func NewBucketRepository(db *sql.DB) *BucketRepository {
	return &BucketRepository{db: db}
}

// Create creates a new bucket
func (r *BucketRepository) Create(ctx context.Context, name string) (*types.Bucket, error) {
	query := `
		INSERT INTO buckets (name, created_at, updated_at)
		VALUES ($1, $2, $3)
		RETURNING name, object_count, total_size, created_at, updated_at
	`

	now := time.Now()
	bucket := &types.Bucket{}

	err := r.db.QueryRowContext(ctx, query, name, now, now).Scan(
		&bucket.Name,
		&bucket.ObjectCount,
		&bucket.TotalSize,
		&bucket.CreatedAt,
		&bucket.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// Get retrieves a bucket by name
func (r *BucketRepository) Get(ctx context.Context, name string) (*types.Bucket, error) {
	query := `
		SELECT name, object_count, total_size, created_at, updated_at
		FROM buckets
		WHERE name = $1
	`

	bucket := &types.Bucket{}

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&bucket.Name,
		&bucket.ObjectCount,
		&bucket.TotalSize,
		&bucket.CreatedAt,
		&bucket.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// List retrieves all buckets
func (r *BucketRepository) List(ctx context.Context) ([]*types.Bucket, error) {
	query := `
		SELECT name, object_count, total_size, created_at, updated_at
		FROM buckets
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []*types.Bucket
	for rows.Next() {
		bucket := &types.Bucket{}
		if err := rows.Scan(
			&bucket.Name,
			&bucket.ObjectCount,
			&bucket.TotalSize,
			&bucket.CreatedAt,
			&bucket.UpdatedAt,
		); err != nil {
			return nil, err
		}
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// Delete deletes a bucket
func (r *BucketRepository) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM buckets WHERE name = $1`

	result, err := r.db.ExecContext(ctx, query, name)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Exists checks if a bucket exists
func (r *BucketRepository) Exists(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM buckets WHERE name = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// IsEmpty checks if a bucket is empty
func (r *BucketRepository) IsEmpty(ctx context.Context, name string) (bool, error) {
	query := `SELECT object_count FROM buckets WHERE name = $1`

	var count int64
	err := r.db.QueryRowContext(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}
