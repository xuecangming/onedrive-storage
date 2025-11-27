package repository

import (
	"database/sql"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// EnhancedVFSRepository handles starred files, trash, and recent files
type EnhancedVFSRepository struct {
	db *sql.DB
}

// NewEnhancedVFSRepository creates a new enhanced VFS repository
func NewEnhancedVFSRepository(db *sql.DB) *EnhancedVFSRepository {
	return &EnhancedVFSRepository{db: db}
}

// ==================== Starred Files ====================

// StarFile adds a file to starred
func (r *EnhancedVFSRepository) StarFile(bucket, fileID, filePath string) error {
	query := `
		INSERT INTO starred_files (bucket, file_id, file_path, starred_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (bucket, file_id) DO UPDATE SET starred_at = NOW()
	`
	_, err := r.db.Exec(query, bucket, fileID, filePath)
	return err
}

// UnstarFile removes a file from starred
func (r *EnhancedVFSRepository) UnstarFile(bucket, fileID string) error {
	query := `DELETE FROM starred_files WHERE bucket = $1 AND file_id = $2`
	_, err := r.db.Exec(query, bucket, fileID)
	return err
}

// IsFileStarred checks if a file is starred
func (r *EnhancedVFSRepository) IsFileStarred(bucket, fileID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM starred_files WHERE bucket = $1 AND file_id = $2)`
	var exists bool
	err := r.db.QueryRow(query, bucket, fileID).Scan(&exists)
	return exists, err
}

// GetStarredFiles returns all starred files for a bucket
func (r *EnhancedVFSRepository) GetStarredFiles(bucket string) ([]types.VFSItem, error) {
	query := `
		SELECT sf.id, vf.name, vf.full_path, 'file' as type, vf.size, vf.mime_type, vf.created_at, vf.updated_at
		FROM starred_files sf
		JOIN virtual_files vf ON sf.file_id = vf.id
		WHERE sf.bucket = $1
		ORDER BY sf.starred_at DESC
	`
	rows, err := r.db.Query(query, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.VFSItem
	for rows.Next() {
		var item types.VFSItem
		var updatedAt sql.NullTime
		err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.Type, &item.Size, &item.MimeType, &item.CreatedAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Time
		}
		item.IsStarred = true
		items = append(items, item)
	}
	return items, nil
}

// GetStarredFileIDs returns IDs of starred files for a bucket
func (r *EnhancedVFSRepository) GetStarredFileIDs(bucket string) (map[string]bool, error) {
	query := `SELECT file_id FROM starred_files WHERE bucket = $1`
	rows, err := r.db.Query(query, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, nil
}

// ==================== Trash/Recycle Bin ====================

// MoveToTrash moves an item to trash
func (r *EnhancedVFSRepository) MoveToTrash(item *types.TrashItem) error {
	query := `
		INSERT INTO trash (bucket, original_type, original_id, original_path, original_name, object_key, size, mime_type, deleted_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW() + INTERVAL '30 days')
	`
	_, err := r.db.Exec(query, item.Bucket, item.OriginalType, item.OriginalID, item.OriginalPath, item.OriginalName, item.ObjectKey, item.Size, item.MimeType)
	return err
}

// GetTrashItems returns all items in trash for a bucket
func (r *EnhancedVFSRepository) GetTrashItems(bucket string) ([]*types.TrashItem, error) {
	query := `
		SELECT id, bucket, original_type, original_id, original_path, original_name, object_key, size, mime_type, deleted_at, expires_at
		FROM trash
		WHERE bucket = $1 AND expires_at > NOW()
		ORDER BY deleted_at DESC
	`
	rows, err := r.db.Query(query, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.TrashItem
	for rows.Next() {
		item := &types.TrashItem{}
		var objectKey sql.NullString
		var size sql.NullInt64
		var mimeType sql.NullString
		err := rows.Scan(&item.ID, &item.Bucket, &item.OriginalType, &item.OriginalID, &item.OriginalPath, &item.OriginalName, &objectKey, &size, &mimeType, &item.DeletedAt, &item.ExpiresAt)
		if err != nil {
			return nil, err
		}
		if objectKey.Valid {
			item.ObjectKey = objectKey.String
		}
		if size.Valid {
			item.Size = size.Int64
		}
		if mimeType.Valid {
			item.MimeType = mimeType.String
		}
		items = append(items, item)
	}
	return items, nil
}

// GetTrashItem returns a single trash item by ID
func (r *EnhancedVFSRepository) GetTrashItem(id string) (*types.TrashItem, error) {
	query := `
		SELECT id, bucket, original_type, original_id, original_path, original_name, object_key, size, mime_type, deleted_at, expires_at
		FROM trash
		WHERE id = $1
	`
	item := &types.TrashItem{}
	var objectKey sql.NullString
	var size sql.NullInt64
	var mimeType sql.NullString
	err := r.db.QueryRow(query, id).Scan(&item.ID, &item.Bucket, &item.OriginalType, &item.OriginalID, &item.OriginalPath, &item.OriginalName, &objectKey, &size, &mimeType, &item.DeletedAt, &item.ExpiresAt)
	if err != nil {
		return nil, err
	}
	if objectKey.Valid {
		item.ObjectKey = objectKey.String
	}
	if size.Valid {
		item.Size = size.Int64
	}
	if mimeType.Valid {
		item.MimeType = mimeType.String
	}
	return item, nil
}

// DeleteFromTrash permanently deletes an item from trash
func (r *EnhancedVFSRepository) DeleteFromTrash(id string) error {
	query := `DELETE FROM trash WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// EmptyTrash removes all items from trash for a bucket
func (r *EnhancedVFSRepository) EmptyTrash(bucket string) (int64, error) {
	query := `DELETE FROM trash WHERE bucket = $1`
	result, err := r.db.Exec(query, bucket)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanupExpiredTrash removes expired items from trash
func (r *EnhancedVFSRepository) CleanupExpiredTrash() ([]*types.TrashItem, error) {
	// First get the items to return (for object cleanup)
	query := `
		SELECT id, bucket, original_type, original_id, original_path, original_name, object_key, size, mime_type, deleted_at, expires_at
		FROM trash
		WHERE expires_at <= NOW()
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.TrashItem
	for rows.Next() {
		item := &types.TrashItem{}
		var objectKey sql.NullString
		var size sql.NullInt64
		var mimeType sql.NullString
		err := rows.Scan(&item.ID, &item.Bucket, &item.OriginalType, &item.OriginalID, &item.OriginalPath, &item.OriginalName, &objectKey, &size, &mimeType, &item.DeletedAt, &item.ExpiresAt)
		if err != nil {
			return nil, err
		}
		if objectKey.Valid {
			item.ObjectKey = objectKey.String
		}
		if size.Valid {
			item.Size = size.Int64
		}
		if mimeType.Valid {
			item.MimeType = mimeType.String
		}
		items = append(items, item)
	}

	// Then delete them
	_, err = r.db.Exec(`DELETE FROM trash WHERE expires_at <= NOW()`)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// ==================== Recent Files ====================

// RecordFileAccess records a file access
func (r *EnhancedVFSRepository) RecordFileAccess(bucket, fileID, filePath, fileName string) error {
	query := `
		INSERT INTO recent_files (bucket, file_id, file_path, file_name, accessed_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (bucket, file_id) DO UPDATE SET accessed_at = NOW(), file_path = EXCLUDED.file_path, file_name = EXCLUDED.file_name
	`
	_, err := r.db.Exec(query, bucket, fileID, filePath, fileName)
	return err
}

// GetRecentFiles returns recently accessed files for a bucket
func (r *EnhancedVFSRepository) GetRecentFiles(bucket string, limit int) ([]types.VFSItem, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT rf.id, vf.name, vf.full_path, 'file' as type, vf.size, vf.mime_type, vf.created_at, vf.updated_at
		FROM recent_files rf
		JOIN virtual_files vf ON rf.file_id = vf.id
		WHERE rf.bucket = $1
		ORDER BY rf.accessed_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, bucket, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.VFSItem
	for rows.Next() {
		var item types.VFSItem
		var updatedAt sql.NullTime
		err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.Type, &item.Size, &item.MimeType, &item.CreatedAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Time
		}
		items = append(items, item)
	}
	return items, nil
}

// CleanupOldRecentFiles removes old recent file entries (keep last N per bucket)
func (r *EnhancedVFSRepository) CleanupOldRecentFiles(bucket string, keepCount int) error {
	query := `
		DELETE FROM recent_files
		WHERE bucket = $1 AND id NOT IN (
			SELECT id FROM recent_files
			WHERE bucket = $1
			ORDER BY accessed_at DESC
			LIMIT $2
		)
	`
	_, err := r.db.Exec(query, bucket, keepCount)
	return err
}

// ==================== Search ====================

// SearchFiles searches for files and directories by name
func (r *EnhancedVFSRepository) SearchFiles(bucket, query string, limit int) ([]types.SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}
	
	searchPattern := "%" + query + "%"
	
	// Search both files and directories
	sqlQuery := `
		SELECT id, name, full_path, 'file' as type, size, mime_type, created_at, 'name' as match_type
		FROM virtual_files
		WHERE bucket = $1 AND (name ILIKE $2 OR full_path ILIKE $2)
		UNION ALL
		SELECT id, name, full_path, 'directory' as type, 0 as size, '' as mime_type, created_at, 'name' as match_type
		FROM virtual_directories
		WHERE bucket = $1 AND (name ILIKE $2 OR full_path ILIKE $2)
		ORDER BY name
		LIMIT $3
	`
	
	rows, err := r.db.Query(sqlQuery, bucket, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []types.SearchResult
	for rows.Next() {
		var result types.SearchResult
		err := rows.Scan(&result.ID, &result.Name, &result.Path, &result.Type, &result.Size, &result.MimeType, &result.CreatedAt, &result.MatchType)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// SearchFilesByType searches for files by type (extension)
func (r *EnhancedVFSRepository) SearchFilesByType(bucket, fileType string, limit int) ([]types.SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}
	
	var mimePattern string
	switch fileType {
	case "image":
		mimePattern = "image/%"
	case "video":
		mimePattern = "video/%"
	case "audio":
		mimePattern = "audio/%"
	case "document":
		mimePattern = "application/pdf%"
	default:
		mimePattern = fileType + "%"
	}
	
	query := `
		SELECT id, name, full_path, 'file' as type, size, mime_type, created_at, 'type' as match_type
		FROM virtual_files
		WHERE bucket = $1 AND mime_type LIKE $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	
	rows, err := r.db.Query(query, bucket, mimePattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []types.SearchResult
	for rows.Next() {
		var result types.SearchResult
		err := rows.Scan(&result.ID, &result.Name, &result.Path, &result.Type, &result.Size, &result.MimeType, &result.CreatedAt, &result.MatchType)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// GetFilesByCreatedDate returns files created within a date range
func (r *EnhancedVFSRepository) GetFilesByCreatedDate(bucket string, from, to time.Time, limit int) ([]types.VFSItem, error) {
	if limit <= 0 {
		limit = 50
	}
	
	query := `
		SELECT id, name, full_path, 'file' as type, size, mime_type, created_at, updated_at
		FROM virtual_files
		WHERE bucket = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
		LIMIT $4
	`
	
	rows, err := r.db.Query(query, bucket, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.VFSItem
	for rows.Next() {
		var item types.VFSItem
		var updatedAt sql.NullTime
		err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.Type, &item.Size, &item.MimeType, &item.CreatedAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Time
		}
		items = append(items, item)
	}
	return items, nil
}
