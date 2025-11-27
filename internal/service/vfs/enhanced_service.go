package vfs

import (
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/repository"
)

// EnhancedService provides enhanced VFS operations (starred, trash, recent, search)
type EnhancedService struct {
	enhancedRepo *repository.EnhancedVFSRepository
	vfsRepo      *repository.VFSRepository
	bucketRepo   *repository.BucketRepository
}

// NewEnhancedService creates a new enhanced VFS service
func NewEnhancedService(enhancedRepo *repository.EnhancedVFSRepository, vfsRepo *repository.VFSRepository, bucketRepo *repository.BucketRepository) *EnhancedService {
	return &EnhancedService{
		enhancedRepo: enhancedRepo,
		vfsRepo:      vfsRepo,
		bucketRepo:   bucketRepo,
	}
}

// ==================== Starred Files ====================

// StarFile stars a file
func (s *EnhancedService) StarFile(bucket, fileID, filePath string) error {
	return s.enhancedRepo.StarFile(bucket, fileID, filePath)
}

// UnstarFile unstars a file
func (s *EnhancedService) UnstarFile(bucket, fileID string) error {
	return s.enhancedRepo.UnstarFile(bucket, fileID)
}

// IsFileStarred checks if a file is starred
func (s *EnhancedService) IsFileStarred(bucket, fileID string) (bool, error) {
	return s.enhancedRepo.IsFileStarred(bucket, fileID)
}

// GetStarredFiles returns all starred files
func (s *EnhancedService) GetStarredFiles(bucket string) ([]types.VFSItem, error) {
	return s.enhancedRepo.GetStarredFiles(bucket)
}

// ==================== Trash ====================

// GetTrashItems returns all items in trash
func (s *EnhancedService) GetTrashItems(bucket string) ([]*types.TrashItem, error) {
	return s.enhancedRepo.GetTrashItems(bucket)
}

// RestoreFromTrash restores an item from trash
func (s *EnhancedService) RestoreFromTrash(trashID string) error {
	// Get the trash item
	item, err := s.enhancedRepo.GetTrashItem(trashID)
	if err != nil {
		return err
	}

	// For files, we need to recreate the virtual file record
	if item.OriginalType == "file" {
		// Check if original path is available
		exists, err := s.vfsRepo.FileExists(item.Bucket, item.OriginalPath)
		if err != nil {
			return err
		}
		if exists {
			// Path is taken, we can't restore
			return &restoreError{message: "original path is no longer available"}
		}

		// Recreate the virtual file (the object should still exist in storage)
		now := time.Now()
		file := &types.VirtualFile{
			ID:        item.OriginalID,
			Bucket:    item.Bucket,
			Name:      item.OriginalName,
			FullPath:  item.OriginalPath,
			ObjectKey: item.ObjectKey,
			Size:      item.Size,
			MimeType:  item.MimeType,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := s.vfsRepo.CreateFile(file); err != nil {
			return err
		}
	}

	// Remove from trash
	return s.enhancedRepo.DeleteFromTrash(trashID)
}

// DeleteFromTrash permanently deletes an item from trash
func (s *EnhancedService) DeleteFromTrash(trashID string) error {
	return s.enhancedRepo.DeleteFromTrash(trashID)
}

// EmptyTrash empties the trash
func (s *EnhancedService) EmptyTrash(bucket string) (int64, error) {
	return s.enhancedRepo.EmptyTrash(bucket)
}

// ==================== Recent Files ====================

// RecordFileAccess records a file access
func (s *EnhancedService) RecordFileAccess(bucket, fileID, filePath, fileName string) error {
	return s.enhancedRepo.RecordFileAccess(bucket, fileID, filePath, fileName)
}

// GetRecentFiles returns recently accessed files
func (s *EnhancedService) GetRecentFiles(bucket string, limit int) ([]types.VFSItem, error) {
	return s.enhancedRepo.GetRecentFiles(bucket, limit)
}

// ==================== Search ====================

// Search searches for files and directories
func (s *EnhancedService) Search(bucket, query string, limit int) ([]types.SearchResult, error) {
	return s.enhancedRepo.SearchFiles(bucket, query, limit)
}

// SearchByType searches for files by type
func (s *EnhancedService) SearchByType(bucket, fileType string, limit int) ([]types.SearchResult, error) {
	return s.enhancedRepo.SearchFilesByType(bucket, fileType, limit)
}

// GetFilesByDateRange returns files within a date range
func (s *EnhancedService) GetFilesByDateRange(bucket string, from, to time.Time, limit int) ([]types.VFSItem, error) {
	return s.enhancedRepo.GetFilesByCreatedDate(bucket, from, to, limit)
}

// restoreError represents a restore error
type restoreError struct {
	message string
}

func (e *restoreError) Error() string {
	return e.message
}
