package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// ObjectRepository handles object data access
type ObjectRepository struct {
	db *sql.DB
}

// NewObjectRepository creates a new object repository
func NewObjectRepository(db *sql.DB) *ObjectRepository {
	return &ObjectRepository{db: db}
}

// Create creates a new object
func (r *ObjectRepository) Create(ctx context.Context, obj *types.Object) error {
	query := `
		INSERT INTO objects (
			bucket, key, account_id, remote_id, remote_path,
			size, etag, mime_type, is_chunked, chunk_count,
			metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	metadataJSON, _ := json.Marshal(obj.Metadata)
	now := time.Now()

	_, err := r.db.ExecContext(ctx, query,
		obj.Bucket, obj.Key, obj.AccountID, obj.RemoteID, obj.RemotePath,
		obj.Size, obj.ETag, obj.MimeType, obj.IsChunked, obj.ChunkCount,
		metadataJSON, now, now,
	)

	if err != nil {
		return err
	}

	obj.CreatedAt = now
	obj.UpdatedAt = now

	return nil
}

// Get retrieves an object by bucket and key
func (r *ObjectRepository) Get(ctx context.Context, bucket, key string) (*types.Object, error) {
	query := `
		SELECT bucket, key, account_id, remote_id, remote_path,
		       size, etag, mime_type, is_chunked, chunk_count,
		       metadata, created_at, updated_at
		FROM objects
		WHERE bucket = $1 AND key = $2
	`

	obj := &types.Object{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, bucket, key).Scan(
		&obj.Bucket, &obj.Key, &obj.AccountID, &obj.RemoteID, &obj.RemotePath,
		&obj.Size, &obj.ETag, &obj.MimeType, &obj.IsChunked, &obj.ChunkCount,
		&metadataJSON, &obj.CreatedAt, &obj.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &obj.Metadata)
	}

	return obj, nil
}

// List retrieves objects in a bucket
func (r *ObjectRepository) List(ctx context.Context, bucket, prefix, marker string, maxKeys int) ([]*types.Object, error) {
	query := `
		SELECT bucket, key, account_id, remote_id, remote_path,
		       size, etag, mime_type, is_chunked, chunk_count,
		       metadata, created_at, updated_at
		FROM objects
		WHERE bucket = $1
		  AND ($2 = '' OR key LIKE $2 || '%')
		  AND ($3 = '' OR key > $3)
		ORDER BY key
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, bucket, prefix, marker, maxKeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*types.Object
	for rows.Next() {
		obj := &types.Object{}
		var metadataJSON []byte

		if err := rows.Scan(
			&obj.Bucket, &obj.Key, &obj.AccountID, &obj.RemoteID, &obj.RemotePath,
			&obj.Size, &obj.ETag, &obj.MimeType, &obj.IsChunked, &obj.ChunkCount,
			&metadataJSON, &obj.CreatedAt, &obj.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &obj.Metadata)
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

// Delete deletes an object
func (r *ObjectRepository) Delete(ctx context.Context, bucket, key string) error {
	query := `DELETE FROM objects WHERE bucket = $1 AND key = $2`

	result, err := r.db.ExecContext(ctx, query, bucket, key)
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

// Exists checks if an object exists
func (r *ObjectRepository) Exists(ctx context.Context, bucket, key string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM objects WHERE bucket = $1 AND key = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, bucket, key).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// UpdateBucketStats updates bucket statistics
func (r *ObjectRepository) UpdateBucketStats(ctx context.Context, bucket string) error {
	query := `
		UPDATE buckets
		SET object_count = (SELECT COUNT(*) FROM objects WHERE bucket = $1),
		    total_size = (SELECT COALESCE(SUM(size), 0) FROM objects WHERE bucket = $1),
		    updated_at = $2
		WHERE name = $1
	`

	_, err := r.db.ExecContext(ctx, query, bucket, time.Now())
	return err
}
