package vfs

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/repository"
	"github.com/xuecangming/onedrive-storage/internal/service/object"
)

// Service handles virtual file system operations
type Service struct {
	vfsRepo    *repository.VFSRepository
	objectSvc  *object.Service
	bucketRepo *repository.BucketRepository
}

// NewService creates a new VFS service
func NewService(vfsRepo *repository.VFSRepository, objectSvc *object.Service, bucketRepo *repository.BucketRepository) *Service {
	return &Service{
		vfsRepo:    vfsRepo,
		objectSvc:  objectSvc,
		bucketRepo: bucketRepo,
	}
}

// UploadFile uploads a file to a virtual path
func (s *Service) UploadFile(bucket, path string, content io.Reader, size int64, mimeType string) (*types.VirtualFile, error) {
	// Validate bucket
	_, err := s.bucketRepo.Get(context.Background(), bucket)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewBucketNotFoundError(bucket)
		}
		return nil, err
	}

	// Normalize path
	path = normalizePath(path)
	if path == "/" || path == "" {
		return nil, errors.NewInvalidRequestError("path cannot be root directory")
	}

	// Check if file already exists
	exists, err := s.vfsRepo.FileExists(bucket, path)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.NewConflictError(fmt.Sprintf("file already exists at path: %s", path))
	}

	// Parse directory and filename
	dirPath, filename := splitPath(path)

	// Ensure directory path exists
	var directoryID *string
	if dirPath != "/" {
		dir, err := s.ensureDirectoryPath(bucket, dirPath)
		if err != nil {
			return nil, err
		}
		directoryID = &dir.ID
	}

	// Generate unique object key
	objectKey := utils.GenerateID()

	// Read content into byte array
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}

	// Upload to object storage
	ctx := context.Background()
	_, err = s.objectSvc.Upload(ctx, bucket, objectKey, data, mimeType)
	if err != nil {
		return nil, err
	}

	// Create virtual file record
	now := time.Now()
	file := &types.VirtualFile{
		ID:          utils.GenerateID(),
		Bucket:      bucket,
		DirectoryID: directoryID,
		Name:        filename,
		FullPath:    path,
		ObjectKey:   objectKey,
		Size:        int64(len(data)), // Use actual data size, not Content-Length
		MimeType:    mimeType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.vfsRepo.CreateFile(file); err != nil {
		// Clean up object if file creation fails
		_ = s.objectSvc.Delete(ctx, bucket, objectKey)
		return nil, err
	}

	return file, nil
}

// GetFile retrieves a file by path
func (s *Service) GetFile(bucket, path string) (*types.VirtualFile, error) {
	path = normalizePath(path)
	file, err := s.vfsRepo.GetFile(bucket, path)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError(fmt.Sprintf("file not found: %s", path))
		}
		return nil, err
	}
	return file, nil
}

// DownloadFile downloads a file by path
func (s *Service) DownloadFile(bucket, path string) ([]byte, *types.VirtualFile, error) {
	file, err := s.GetFile(bucket, path)
	if err != nil {
		return nil, nil, err
	}

	// Download from object storage
	ctx := context.Background()
	_, data, err := s.objectSvc.Download(ctx, bucket, file.ObjectKey)
	if err != nil {
		return nil, nil, err
	}

	return data, file, nil
}

// ListDirectory lists contents of a directory
func (s *Service) ListDirectory(bucket, path string, recursive bool) ([]types.VFSItem, error) {
	// Validate bucket
	ctx := context.Background()
	_, err := s.bucketRepo.Get(ctx, bucket)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewBucketNotFoundError(bucket)
		}
		return nil, err
	}

	path = normalizePath(path)

	// Get directory
	var directoryID *string
	if path != "/" {
		dir, err := s.vfsRepo.GetDirectory(bucket, path)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.NewNotFoundError(fmt.Sprintf("directory not found: %s", path))
			}
			return nil, err
		}
		directoryID = &dir.ID
	}

	if recursive {
		return s.listRecursive(bucket, path)
	}

	// List immediate children
	return s.vfsRepo.ListDirectoryContents(bucket, directoryID)
}

// listRecursive lists all items recursively under a path
func (s *Service) listRecursive(bucket, path string) ([]types.VFSItem, error) {
	var items []types.VFSItem

	// Get all directories under this path
	dirs, err := s.vfsRepo.ListDirectoriesByPath(bucket, path)
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		item := types.VFSItem{
			ID:        dir.ID,
			Name:      dir.Name,
			Path:      dir.FullPath,
			Type:      "directory",
			CreatedAt: dir.CreatedAt,
		}
		items = append(items, item)
	}

	// Get all files under this path
	files, err := s.vfsRepo.ListFilesByDirectory(bucket, path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		item := types.VFSItem{
			ID:        file.ID,
			Name:      file.Name,
			Path:      file.FullPath,
			Type:      "file",
			Size:      file.Size,
			MimeType:  file.MimeType,
			CreatedAt: file.CreatedAt,
			UpdatedAt: &file.UpdatedAt,
		}
		items = append(items, item)
	}

	return items, nil
}

// CreateDirectory creates a directory
func (s *Service) CreateDirectory(bucket, path string) (*types.VirtualDirectory, error) {
	// Validate bucket
	ctx := context.Background()
	_, err := s.bucketRepo.Get(ctx, bucket)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewBucketNotFoundError(bucket)
		}
		return nil, err
	}

	path = normalizePath(path)
	if path == "/" {
		return nil, errors.NewInvalidRequestError("cannot create root directory")
	}

	// Check if directory already exists
	exists, err := s.vfsRepo.DirectoryExists(bucket, path)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.NewConflictError(fmt.Sprintf("directory already exists: %s", path))
	}

	return s.ensureDirectoryPath(bucket, path)
}

// DeleteFile deletes a file
func (s *Service) DeleteFile(bucket, path string) error {
	path = normalizePath(path)

	file, err := s.vfsRepo.GetFile(bucket, path)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewNotFoundError(fmt.Sprintf("file not found: %s", path))
		}
		return err
	}

	// Delete virtual file record
	if err := s.vfsRepo.DeleteFile(file.ID); err != nil {
		return err
	}

	// Delete from object storage
	ctx := context.Background()
	if err := s.objectSvc.Delete(ctx, bucket, file.ObjectKey); err != nil {
		// Log error but don't fail - virtual file is already deleted
		// In production, this should use proper logging framework
		fmt.Printf("Warning: failed to delete object %s from storage: %v\n", file.ObjectKey, err)
	}

	return nil
}

// DeleteDirectory deletes a directory
func (s *Service) DeleteDirectory(bucket, path string, recursive bool) error {
	path = normalizePath(path)
	if path == "/" {
		return errors.NewInvalidRequestError("cannot delete root directory")
	}

	dir, err := s.vfsRepo.GetDirectory(bucket, path)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewNotFoundError(fmt.Sprintf("directory not found: %s", path))
		}
		return err
	}

	// Check if directory has children
	childCount, err := s.vfsRepo.CountDirectoryChildren(dir.ID)
	if err != nil {
		return err
	}

	if childCount > 0 && !recursive {
		return errors.NewConflictError("directory is not empty, use recursive=true to delete")
	}

	var objectKeysToDelete []string
	if recursive && childCount > 0 {
		// Get all files before deleting (so we can clean up objects after)
		searchPath := strings.TrimSuffix(path, "/")
		files, err := s.vfsRepo.ListFilesByDirectory(bucket, searchPath)
		if err != nil {
			return fmt.Errorf("failed to list files for deletion: %w", err)
		}

		// Collect object keys
		for _, file := range files {
			objectKeysToDelete = append(objectKeysToDelete, file.ObjectKey)
		}
	}

	// Delete the directory itself (this will cascade delete all virtual_files and subdirectories)
	if err := s.vfsRepo.DeleteDirectory(dir.ID); err != nil {
		return err
	}

	// Now delete the objects from storage (after virtual_files are deleted)
	ctx := context.Background()
	for _, objectKey := range objectKeysToDelete {
		if err := s.objectSvc.Delete(ctx, bucket, objectKey); err != nil {
			// Log error but continue - virtual files are already deleted
			fmt.Printf("Warning: failed to delete object %s from storage: %v\n", objectKey, err)
		}
	}

	return nil
}

// MoveFile moves or renames a file
func (s *Service) MoveFile(bucket, source, destination string) (*types.VirtualFile, error) {
	source = normalizePath(source)
	destination = normalizePath(destination)

	if source == destination {
		return nil, errors.NewInvalidRequestError("source and destination are the same")
	}

	// Get source file
	file, err := s.vfsRepo.GetFile(bucket, source)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError(fmt.Sprintf("source file not found: %s", source))
		}
		return nil, err
	}

	// Check if destination already exists
	exists, err := s.vfsRepo.FileExists(bucket, destination)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.NewConflictError(fmt.Sprintf("destination file already exists: %s", destination))
	}

	// Parse destination directory and filename
	destDirPath, destFilename := splitPath(destination)

	// Ensure destination directory exists
	var destDirID *string
	if destDirPath != "/" {
		dir, err := s.ensureDirectoryPath(bucket, destDirPath)
		if err != nil {
			return nil, err
		}
		destDirID = &dir.ID
	}

	// Update file
	file.DirectoryID = destDirID
	file.Name = destFilename
	file.FullPath = destination
	file.UpdatedAt = time.Now()

	if err := s.vfsRepo.UpdateFile(file); err != nil {
		return nil, err
	}

	return file, nil
}

// MoveDirectory moves or renames a directory
func (s *Service) MoveDirectory(bucket, source, destination string) (*types.VirtualDirectory, error) {
	source = normalizePath(source)
	destination = normalizePath(destination)

	if source == "/" || destination == "/" {
		return nil, errors.NewInvalidRequestError("cannot move root directory")
	}

	if source == destination {
		return nil, errors.NewInvalidRequestError("source and destination are the same")
	}

	// Check if destination is a subdirectory of source
	if strings.HasPrefix(destination, source+"/") {
		return nil, errors.NewInvalidRequestError("cannot move directory into itself")
	}

	// Get source directory
	dir, err := s.vfsRepo.GetDirectory(bucket, source)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError(fmt.Sprintf("source directory not found: %s", source))
		}
		return nil, err
	}

	// Check if destination already exists
	exists, err := s.vfsRepo.DirectoryExists(bucket, destination)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.NewConflictError(fmt.Sprintf("destination directory already exists: %s", destination))
	}

	// Parse destination parent directory and name
	destParentPath, destName := splitPath(destination)

	// Ensure destination parent directory exists
	var destParentID *string
	if destParentPath != "/" {
		parentDir, err := s.ensureDirectoryPath(bucket, destParentPath)
		if err != nil {
			return nil, err
		}
		destParentID = &parentDir.ID
	}

	// Update all subdirectories and files with new paths
	if err := s.updatePathsAfterMove(bucket, source, destination); err != nil {
		return nil, err
	}

	// Update the directory itself
	dir.ParentID = destParentID
	dir.Name = destName
	dir.FullPath = destination

	if err := s.vfsRepo.UpdateDirectory(dir); err != nil {
		return nil, err
	}

	return dir, nil
}

// updatePathsAfterMove updates all paths under a moved directory
func (s *Service) updatePathsAfterMove(bucket, oldPath, newPath string) error {
	// Ensure paths don't have trailing slashes for consistent replacement
	oldPath = strings.TrimSuffix(oldPath, "/")
	newPath = strings.TrimSuffix(newPath, "/")

	// Get all subdirectories
	dirs, err := s.vfsRepo.ListDirectoriesByPath(bucket, oldPath+"/")
	if err != nil {
		return err
	}

	// Update subdirectory paths
	for _, dir := range dirs {
		newFullPath := strings.Replace(dir.FullPath, oldPath, newPath, 1)
		dir.FullPath = newFullPath
		if err := s.vfsRepo.UpdateDirectory(dir); err != nil {
			return err
		}
	}

	// Get all files
	files, err := s.vfsRepo.ListFilesByDirectory(bucket, oldPath+"/")
	if err != nil {
		return err
	}

	// Update file paths
	for _, file := range files {
		newFullPath := strings.Replace(file.FullPath, oldPath, newPath, 1)
		file.FullPath = newFullPath
		file.UpdatedAt = time.Now()
		if err := s.vfsRepo.UpdateFile(file); err != nil {
			return err
		}
	}

	return nil
}

// ensureDirectoryPath ensures all directories in a path exist, creating them if necessary
func (s *Service) ensureDirectoryPath(bucket, path string) (*types.VirtualDirectory, error) {
	path = normalizePath(path)
	if path == "/" {
		return nil, nil
	}

	// Check if directory already exists
	dir, err := s.vfsRepo.GetDirectory(bucket, path)
	if err == nil {
		return dir, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Directory doesn't exist, create it and all parent directories
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var currentPath string
	var parentID *string

	for _, part := range parts {
		if currentPath == "" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Check if this level exists
		existingDir, err := s.vfsRepo.GetDirectory(bucket, currentPath)
		if err == nil {
			parentID = &existingDir.ID
			dir = existingDir
			continue
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		// Create this level
		newDir := &types.VirtualDirectory{
			ID:        utils.GenerateID(),
			Bucket:    bucket,
			ParentID:  parentID,
			Name:      part,
			FullPath:  currentPath,
			CreatedAt: time.Now(),
		}

		if err := s.vfsRepo.CreateDirectory(newDir); err != nil {
			return nil, err
		}

		parentID = &newDir.ID
		dir = newDir
	}

	return dir, nil
}

// normalizePath normalizes a virtual path
func normalizePath(path string) string {
	// Clean the path
	path = filepath.Clean("/" + path)

	// Ensure it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Ensure directories end with /
	// Files should not end with /

	return path
}

// splitPath splits a path into directory and filename
func splitPath(path string) (string, string) {
	path = strings.TrimSuffix(path, "/")
	dir, file := filepath.Split(path)

	if dir == "" {
		dir = "/"
	} else {
		dir = strings.TrimSuffix(dir, "/")
	}

	return dir, file
}
