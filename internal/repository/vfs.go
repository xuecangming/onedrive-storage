package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// VFSRepository handles virtual file system data access
type VFSRepository struct {
	db *sql.DB
}

// NewVFSRepository creates a new VFS repository
func NewVFSRepository(db *sql.DB) *VFSRepository {
	return &VFSRepository{db: db}
}

// CreateDirectory creates a new virtual directory
func (r *VFSRepository) CreateDirectory(dir *types.VirtualDirectory) error {
	query := `
		INSERT INTO virtual_directories (id, bucket, parent_id, name, full_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query, dir.ID, dir.Bucket, dir.ParentID, dir.Name, dir.FullPath, dir.CreatedAt)
	return err
}

// GetDirectory retrieves a directory by bucket and full path
func (r *VFSRepository) GetDirectory(bucket, fullPath string) (*types.VirtualDirectory, error) {
	query := `
		SELECT id, bucket, parent_id, name, full_path, created_at
		FROM virtual_directories
		WHERE bucket = $1 AND full_path = $2
	`
	dir := &types.VirtualDirectory{}
	var parentID sql.NullString
	err := r.db.QueryRow(query, bucket, fullPath).Scan(
		&dir.ID, &dir.Bucket, &parentID, &dir.Name, &dir.FullPath, &dir.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		dir.ParentID = &parentID.String
	}
	return dir, nil
}

// GetDirectoryByID retrieves a directory by its ID
func (r *VFSRepository) GetDirectoryByID(id string) (*types.VirtualDirectory, error) {
	query := `
		SELECT id, bucket, parent_id, name, full_path, created_at
		FROM virtual_directories
		WHERE id = $1
	`
	dir := &types.VirtualDirectory{}
	var parentID sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&dir.ID, &dir.Bucket, &parentID, &dir.Name, &dir.FullPath, &dir.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		dir.ParentID = &parentID.String
	}
	return dir, nil
}

// ListDirectoryContents lists all files and subdirectories in a directory
func (r *VFSRepository) ListDirectoryContents(bucket string, parentID *string) ([]types.VFSItem, error) {
	var items []types.VFSItem

	// List subdirectories
	dirQuery := `
		SELECT id, name, full_path, 'directory' as type, 0 as size, '' as mime_type, created_at, created_at as updated_at
		FROM virtual_directories
		WHERE bucket = $1 AND ($2::uuid IS NULL AND parent_id IS NULL OR parent_id = $2)
		ORDER BY name
	`
	rows, err := r.db.Query(dirQuery, bucket, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item types.VFSItem
		var updatedAt time.Time
		err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.Type, &item.Size, &item.MimeType, &item.CreatedAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	// List files
	fileQuery := `
		SELECT id, name, full_path, 'file' as type, size, mime_type, created_at, updated_at
		FROM virtual_files
		WHERE bucket = $1 AND ($2::uuid IS NULL AND directory_id IS NULL OR directory_id = $2)
		ORDER BY name
	`
	rows, err = r.db.Query(fileQuery, bucket, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item types.VFSItem
		err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.Type, &item.Size, &item.MimeType, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// DeleteDirectory deletes a directory
func (r *VFSRepository) DeleteDirectory(id string) error {
	query := `DELETE FROM virtual_directories WHERE id = $1`
	result, err := r.db.Exec(query, id)
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

// CreateFile creates a new virtual file
func (r *VFSRepository) CreateFile(file *types.VirtualFile) error {
	query := `
		INSERT INTO virtual_files (id, bucket, directory_id, name, full_path, object_key, size, mime_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(query, file.ID, file.Bucket, file.DirectoryID, file.Name, file.FullPath, file.ObjectKey, file.Size, file.MimeType, file.CreatedAt, file.UpdatedAt)
	return err
}

// GetFile retrieves a file by bucket and full path
func (r *VFSRepository) GetFile(bucket, fullPath string) (*types.VirtualFile, error) {
	query := `
		SELECT id, bucket, directory_id, name, full_path, object_key, size, mime_type, created_at, updated_at
		FROM virtual_files
		WHERE bucket = $1 AND full_path = $2
	`
	file := &types.VirtualFile{}
	var directoryID sql.NullString
	err := r.db.QueryRow(query, bucket, fullPath).Scan(
		&file.ID, &file.Bucket, &directoryID, &file.Name, &file.FullPath, &file.ObjectKey, &file.Size, &file.MimeType, &file.CreatedAt, &file.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if directoryID.Valid {
		file.DirectoryID = &directoryID.String
	}
	return file, nil
}

// GetFileByID retrieves a file by its ID
func (r *VFSRepository) GetFileByID(id string) (*types.VirtualFile, error) {
	query := `
		SELECT id, bucket, directory_id, name, full_path, object_key, size, mime_type, created_at, updated_at
		FROM virtual_files
		WHERE id = $1
	`
	file := &types.VirtualFile{}
	var directoryID sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&file.ID, &file.Bucket, &directoryID, &file.Name, &file.FullPath, &file.ObjectKey, &file.Size, &file.MimeType, &file.CreatedAt, &file.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if directoryID.Valid {
		file.DirectoryID = &directoryID.String
	}
	return file, nil
}

// DeleteFile deletes a virtual file
func (r *VFSRepository) DeleteFile(id string) error {
	query := `DELETE FROM virtual_files WHERE id = $1`
	result, err := r.db.Exec(query, id)
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

// UpdateFile updates a virtual file's metadata
func (r *VFSRepository) UpdateFile(file *types.VirtualFile) error {
	query := `
		UPDATE virtual_files
		SET directory_id = $1, name = $2, full_path = $3, updated_at = $4
		WHERE id = $5
	`
	result, err := r.db.Exec(query, file.DirectoryID, file.Name, file.FullPath, file.UpdatedAt, file.ID)
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

// UpdateDirectory updates a virtual directory's metadata
func (r *VFSRepository) UpdateDirectory(dir *types.VirtualDirectory) error {
	query := `
		UPDATE virtual_directories
		SET parent_id = $1, name = $2, full_path = $3
		WHERE id = $4
	`
	result, err := r.db.Exec(query, dir.ParentID, dir.Name, dir.FullPath, dir.ID)
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

// CountDirectoryChildren counts files and subdirectories in a directory
func (r *VFSRepository) CountDirectoryChildren(id string) (int, error) {
	var count int

	// Count subdirectories
	dirQuery := `SELECT COUNT(*) FROM virtual_directories WHERE parent_id = $1`
	err := r.db.QueryRow(dirQuery, id).Scan(&count)
	if err != nil {
		return 0, err
	}

	// Count files
	fileQuery := `SELECT COUNT(*) FROM virtual_files WHERE directory_id = $1`
	var fileCount int
	err = r.db.QueryRow(fileQuery, id).Scan(&fileCount)
	if err != nil {
		return 0, err
	}

	return count + fileCount, nil
}

// ListDirectoriesByPath lists all directories matching a path prefix
func (r *VFSRepository) ListDirectoriesByPath(bucket, pathPrefix string) ([]*types.VirtualDirectory, error) {
	query := `
		SELECT id, bucket, parent_id, name, full_path, created_at
		FROM virtual_directories
		WHERE bucket = $1 AND full_path LIKE $2
		ORDER BY full_path
	`
	rows, err := r.db.Query(query, bucket, pathPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dirs []*types.VirtualDirectory
	for rows.Next() {
		dir := &types.VirtualDirectory{}
		var parentID sql.NullString
		err := rows.Scan(&dir.ID, &dir.Bucket, &parentID, &dir.Name, &dir.FullPath, &dir.CreatedAt)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			dir.ParentID = &parentID.String
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

// ListFilesByDirectory lists all files in a directory and its subdirectories
func (r *VFSRepository) ListFilesByDirectory(bucket, pathPrefix string) ([]*types.VirtualFile, error) {
	query := `
		SELECT id, bucket, directory_id, name, full_path, object_key, size, mime_type, created_at, updated_at
		FROM virtual_files
		WHERE bucket = $1 AND full_path LIKE $2
		ORDER BY full_path
	`
	rows, err := r.db.Query(query, bucket, pathPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*types.VirtualFile
	for rows.Next() {
		file := &types.VirtualFile{}
		var directoryID sql.NullString
		err := rows.Scan(&file.ID, &file.Bucket, &directoryID, &file.Name, &file.FullPath, &file.ObjectKey, &file.Size, &file.MimeType, &file.CreatedAt, &file.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if directoryID.Valid {
			file.DirectoryID = &directoryID.String
		}
		files = append(files, file)
	}
	return files, nil
}

// DirectoryExists checks if a directory exists
func (r *VFSRepository) DirectoryExists(bucket, fullPath string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM virtual_directories WHERE bucket = $1 AND full_path = $2)`
	var exists bool
	err := r.db.QueryRow(query, bucket, fullPath).Scan(&exists)
	return exists, err
}

// FileExists checks if a file exists
func (r *VFSRepository) FileExists(bucket, fullPath string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM virtual_files WHERE bucket = $1 AND full_path = $2)`
	var exists bool
	err := r.db.QueryRow(query, bucket, fullPath).Scan(&exists)
	return exists, err
}

// BeginTx starts a database transaction
func (r *VFSRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// DeleteDirectoryTx deletes a directory within a transaction
func (r *VFSRepository) DeleteDirectoryTx(tx *sql.Tx, id string) error {
	query := `DELETE FROM virtual_directories WHERE id = $1`
	result, err := tx.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("directory not found")
	}
	return nil
}

// DeleteFileTx deletes a file within a transaction
func (r *VFSRepository) DeleteFileTx(tx *sql.Tx, id string) error {
	query := `DELETE FROM virtual_files WHERE id = $1`
	result, err := tx.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found")
	}
	return nil
}
