package object

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/core/loadbalancer"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/onedrive"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/storage"
	"github.com/xuecangming/onedrive-storage/internal/repository"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

const DefaultChunkSize = 10 * 1024 * 1024 // 10MB

// Service provides object storage operations
type Service struct {
	objectRepo     *repository.ObjectRepository
	bucketRepo     *repository.BucketRepository
	accountService *account.Service
	balancer       *loadbalancer.Balancer
	useOneDrive    bool                   // Flag to enable/disable OneDrive
	localStorage   *storage.LocalStorage  // Local file storage
}

// NewService creates a new object service with local file storage
func NewService(objectRepo *repository.ObjectRepository, bucketRepo *repository.BucketRepository) *Service {
	// Initialize local storage
	localStorage, err := storage.NewLocalStorage("./data/storage")
	if err != nil {
		log.Printf("Warning: failed to initialize local storage: %v, using memory fallback", err)
	}
	
	return &Service{
		objectRepo:   objectRepo,
		bucketRepo:   bucketRepo,
		useOneDrive:  false,
		localStorage: localStorage,
		balancer:     loadbalancer.NewBalancer(loadbalancer.StrategyLeastUsed),
	}
}

// NewServiceWithOneDrive creates a new object service with OneDrive integration
func NewServiceWithOneDrive(objectRepo *repository.ObjectRepository, bucketRepo *repository.BucketRepository, accountService *account.Service) *Service {
	// Initialize local storage as fallback
	localStorage, _ := storage.NewLocalStorage("./data/storage")
	
	return &Service{
		objectRepo:     objectRepo,
		bucketRepo:     bucketRepo,
		accountService: accountService,
		balancer:       loadbalancer.NewBalancer(loadbalancer.StrategyLeastUsed),
		useOneDrive:    true,
		localStorage:   localStorage,
	}
}

// Upload uploads an object
func (s *Service) Upload(ctx context.Context, bucket, key string, content io.Reader, size int64, mimeType string) (*types.Object, error) {
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

	// If file is small enough, upload as single object
	if size <= DefaultChunkSize {
		data, err := io.ReadAll(content)
		if err != nil {
			return nil, err
		}
		return s.uploadSingle(ctx, bucket, key, data, mimeType)
	}

	// Large file: Chunked upload
	return s.uploadChunked(ctx, bucket, key, content, size, mimeType)
}

// InitiateMultipartUpload starts a new multipart upload
func (s *Service) InitiateMultipartUpload(ctx context.Context, bucket, key, mimeType string) (string, error) {
	// Validate bucket and key
	if !utils.ValidateBucketName(bucket) {
		return "", errors.InvalidBucket(bucket)
	}
	if !utils.ValidateObjectKey(key) {
		return "", errors.InvalidKey(key)
	}

	// Check if object already exists
	exists, err := s.objectRepo.Exists(ctx, bucket, key)
	if err != nil {
		return "", err
	}
	if exists {
		return "", errors.NewConflictError(fmt.Sprintf("object %s/%s already exists", bucket, key))
	}

	// In a real implementation, we would create a record in a multipart_uploads table
	// For now, we just return the key as the upload ID
	
	// Create placeholder object to satisfy FK constraint for chunks
	obj := &types.Object{
		Bucket:     bucket,
		Key:        key,
		Size:       0,
		MimeType:   mimeType,
		IsChunked:  true,
		ChunkCount: 0,
		Metadata:   make(map[string]string),
		AccountID:  "00000000-0000-0000-0000-000000000000", // Placeholder
	}

	if err := s.objectRepo.Create(ctx, obj); err != nil {
		return "", errors.InternalError(err.Error())
	}

	return key, nil
}

// UploadPart uploads a part of a multipart upload
func (s *Service) UploadPart(ctx context.Context, bucket, key string, partNumber int, data []byte) (*types.ObjectChunk, error) {
	// Reuse uploadOneChunk logic
	// Note: uploadOneChunk saves to DB
	err := s.uploadOneChunk(ctx, bucket, key, partNumber, data)
	if err != nil {
		return nil, err
	}

	// Retrieve the created chunk to return it
	// This is a bit inefficient but reuses existing logic
	chunks, err := s.objectRepo.GetChunks(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	
	// Find the chunk we just added
	for _, chunk := range chunks {
		if chunk.ChunkIndex == partNumber {
			return chunk, nil
		}
	}
	
	return nil, errors.InternalError("failed to retrieve uploaded chunk")
}

// CompleteMultipartUpload completes a multipart upload
func (s *Service) CompleteMultipartUpload(ctx context.Context, bucket, key string, totalSize int64, mimeType string) (*types.Object, error) {
	// Verify chunks exist
	chunks, err := s.objectRepo.GetChunks(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, errors.NewInvalidRequestError("no chunks found for upload")
	}

	// Create object metadata
	obj := &types.Object{
		Bucket:     bucket,
		Key:        key,
		Size:       totalSize,
		MimeType:   mimeType,
		IsChunked:  true,
		ChunkCount: len(chunks),
		Metadata:   make(map[string]string),
		AccountID:  "00000000-0000-0000-0000-000000000000", // Distributed
	}

	// Update database (object was created in InitiateMultipartUpload)
	if err := s.objectRepo.Update(ctx, obj); err != nil {
		return nil, errors.InternalError(err.Error())
	}

	// Update bucket stats
	s.objectRepo.UpdateBucketStats(ctx, bucket)

	return obj, nil
}

// GetThumbnail retrieves a thumbnail for an object
func (s *Service) GetThumbnail(ctx context.Context, bucket, key string, size string) ([]byte, string, error) {
	// Get object metadata
	obj, err := s.objectRepo.Get(ctx, bucket, key)
	if err != nil {
		return nil, "", err
	}

	// If object is not on OneDrive, we can't generate thumbnail easily (unless we implement local thumbnailer)
	if obj.RemoteID == "" || obj.RemoteID == "local-storage" {
		return nil, "", errors.NewInvalidRequestError("thumbnails not supported for local storage")
	}

	// Get account
	account, err := s.accountService.Get(ctx, obj.AccountID)
	if err != nil {
		return nil, "", err
	}

	// Ensure token is valid
	if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
		return nil, "", err
	}
	account, _ = s.accountService.Get(ctx, account.ID)

	// Get thumbnail from OneDrive
	client := onedrive.NewClient(account.AccessToken)
	return client.GetThumbnail(ctx, obj.RemoteID, size)
}

func (s *Service) uploadSingle(ctx context.Context, bucket, key string, data []byte, mimeType string) (*types.Object, error) {
	// Calculate ETag (MD5 hash)
	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	var accountID, remoteID, remotePath string
	uploadedToOneDrive := false

	// Try to upload to OneDrive if enabled
	if s.useOneDrive && s.accountService != nil {
		// Get active accounts
		accounts, err := s.accountService.GetActiveAccounts(ctx)
		if err != nil {
			log.Printf("Failed to get active accounts: %v", err)
		} else if len(accounts) == 0 {
			log.Printf("No active OneDrive accounts available, falling back to local storage")
		} else {
			// Select account using load balancer
			account, err := s.balancer.SelectAccount(ctx, accounts, int64(len(data)))
			if err != nil {
				log.Printf("Failed to select account: %v, falling back to local storage", err)
			} else {
				// Ensure token is valid
				if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
					log.Printf("Failed to ensure token valid for account %s: %v", account.ID, err)
				} else {
					// Get fresh account with updated token
					account, err = s.accountService.Get(ctx, account.ID)
					if err != nil {
						log.Printf("Failed to get fresh account %s: %v", account.ID, err)
					} else {
						// Create OneDrive client
						client := onedrive.NewClient(account.AccessToken)

						// Upload file to OneDrive
						path := fmt.Sprintf("%s/%s", bucket, key)
						log.Printf("Uploading file to OneDrive: %s (size: %d bytes)", path, len(data))
						item, err := client.UploadSmallFile(ctx, path, data)
						if err != nil {
							log.Printf("Failed to upload to OneDrive: %v", err)
						} else {
							log.Printf("Successfully uploaded to OneDrive: %s (ID: %s)", path, item.ID)
							accountID = account.ID
							remoteID = item.ID
							remotePath = path
							uploadedToOneDrive = true
						}
					}
				}
			}
		}
	}

	// Fallback to local storage if OneDrive upload failed or not enabled
	if !uploadedToOneDrive {
		if s.localStorage != nil {
			// Store data in local file system
			filePath, err := s.localStorage.Store(bucket, key, data)
			if err != nil {
				return nil, errors.InternalError(fmt.Sprintf("failed to store file: %v", err))
			}

			// Use dummy account for local storage
			accountID = "00000000-0000-0000-0000-000000000000"
			remoteID = "local-storage"
			remotePath = filePath
		} else {
			return nil, errors.InternalError("no storage backend available")
		}
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

// ListParts lists uploaded parts for a multipart upload
func (s *Service) ListParts(ctx context.Context, bucket, key string) ([]*types.ObjectChunk, error) {
	// Verify chunks exist
	chunks, err := s.objectRepo.GetChunks(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

// AbortMultipartUpload aborts a multipart upload and deletes uploaded parts
func (s *Service) AbortMultipartUpload(ctx context.Context, bucket, key string) error {
	// Get all chunks
	chunks, err := s.objectRepo.GetChunks(ctx, bucket, key)
	if err != nil {
		return err
	}

	// Delete chunks from OneDrive
	for _, chunk := range chunks {
		// Get account
		account, err := s.accountService.Get(ctx, chunk.AccountID)
		if err != nil {
			log.Printf("Warning: failed to get account %s for chunk deletion: %v", chunk.AccountID, err)
			continue
		}

		// Ensure token is valid
		if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
			log.Printf("Warning: failed to refresh token for account %s: %v", account.ID, err)
			continue
		}
		account, _ = s.accountService.Get(ctx, account.ID)

		// Delete from OneDrive
		client := onedrive.NewClient(account.AccessToken)
		if err := client.DeleteFile(ctx, chunk.RemoteID); err != nil {
			log.Printf("Warning: failed to delete chunk %s from OneDrive: %v", chunk.RemoteID, err)
		}
	}

	// Delete chunks from DB
	// Note: We need a method in repo to delete all chunks for a key
	// For now, we can iterate or add a new method.
	// Let's add DeleteChunks method to repo later.
	// For now, we assume the user will implement it or we just leave them as orphaned?
	// No, we should delete them.
	
	// Since we don't have DeleteChunks in repo yet, let's add it or use raw SQL if possible?
	// We can't modify repo easily without reading it again.
	// But wait, I can modify repo.
	
	return s.objectRepo.DeleteChunks(ctx, bucket, key)
}

func (s *Service) uploadChunked(ctx context.Context, bucket, key string, content io.Reader, size int64, mimeType string) (*types.Object, error) {
	chunkCount := int(math.Ceil(float64(size) / float64(DefaultChunkSize)))

	buffer := make([]byte, DefaultChunkSize)
	for i := 0; i < chunkCount; i++ {
		bytesRead, err := io.ReadFull(content, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		if bytesRead == 0 {
			break
		}

		chunkData := buffer[:bytesRead]

		// Upload chunk
		err = s.uploadOneChunk(ctx, bucket, key, i, chunkData)
		if err != nil {
			return nil, err
		}
	}

	// Create object metadata
	obj := &types.Object{
		Bucket:     bucket,
		Key:        key,
		Size:       size,
		MimeType:   mimeType,
		IsChunked:  true,
		ChunkCount: chunkCount,
		Metadata:   make(map[string]string),
		AccountID:  "00000000-0000-0000-0000-000000000000", // Distributed
	}

	// Save to database
	if err := s.objectRepo.Create(ctx, obj); err != nil {
		return nil, errors.InternalError(err.Error())
	}

	// Update bucket stats
	s.objectRepo.UpdateBucketStats(ctx, bucket)

	return obj, nil
}

func (s *Service) uploadOneChunk(ctx context.Context, bucket, key string, index int, data []byte) error {
	// Get active accounts
	accounts, err := s.accountService.GetActiveAccounts(ctx)
	if err != nil || len(accounts) == 0 {
		return errors.InternalError("no active accounts available for chunk upload")
	}

	// Select account using load balancer
	account, err := s.balancer.SelectAccount(ctx, accounts, int64(len(data)))
	if err != nil {
		return err
	}

	// Ensure token is valid
	if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
		return err
	}
	account, _ = s.accountService.Get(ctx, account.ID)

	// Upload to OneDrive
	client := onedrive.NewClient(account.AccessToken)
	remotePath := fmt.Sprintf("%s/%s_part%d", bucket, key, index)
	item, err := client.UploadSmallFile(ctx, remotePath, data)
	if err != nil {
		return err
	}

	// Save chunk metadata
	chunk := &types.ObjectChunk{
		ID:         utils.GenerateID(),
		Bucket:     bucket,
		Key:        key,
		ChunkIndex: index,
		AccountID:  account.ID,
		RemoteID:   item.ID,
		RemotePath: remotePath,
		ChunkSize:  int64(len(data)),
		Status:     "active",
	}
	return s.objectRepo.CreateChunk(ctx, chunk)
}

// ReadSeekCloser combines Reader, Seeker and Closer
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type seekCloserWrapper struct {
	io.ReadSeeker
}

func (w *seekCloserWrapper) Close() error { return nil }

// Download downloads an object
func (s *Service) Download(ctx context.Context, bucket, key string) (*types.Object, ReadSeekCloser, error) {
	// Get object metadata
	obj, err := s.objectRepo.Get(ctx, bucket, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.ObjectNotFound(bucket, key)
		}
		return nil, nil, errors.InternalError(err.Error())
	}

	if obj.IsChunked {
		chunks, err := s.objectRepo.GetChunks(ctx, bucket, key)
		if err != nil {
			return nil, nil, errors.InternalError(err.Error())
		}
		return obj, &ChunkReader{
			ctx:       ctx,
			service:   s,
			chunks:    chunks,
			totalSize: obj.Size,
		}, nil
	}

	reader, err := s.downloadSingle(ctx, obj)
	if err != nil {
		return nil, nil, err
	}
	return obj, reader, nil
}

func (s *Service) downloadSingle(ctx context.Context, obj *types.Object) (ReadSeekCloser, error) {
	// Download from OneDrive if enabled and not using dummy account
	if s.useOneDrive && s.accountService != nil && obj.AccountID != "00000000-0000-0000-0000-000000000000" {
		// Get account
		account, err := s.accountService.Get(ctx, obj.AccountID)
		if err != nil {
			return nil, err
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

		// Download file from OneDrive
		data, err := client.DownloadFile(ctx, obj.RemoteID)
		if err != nil {
			return nil, errors.UpstreamError(err.Error())
		}
		return &seekCloserWrapper{bytes.NewReader(data)}, nil
	} else if s.localStorage != nil {
		// Retrieve data from local file storage
		data, err := s.localStorage.Retrieve(obj.Bucket, obj.Key)
		if err != nil {
			return nil, errors.ObjectNotFound(obj.Bucket, obj.Key)
		}
		return &seekCloserWrapper{bytes.NewReader(data)}, nil
	}

	return nil, errors.InternalError("no storage backend available")
}

type ChunkReader struct {
	ctx           context.Context
	service       *Service
	chunks        []*types.ObjectChunk
	currentIdx    int
	currentReader io.ReadSeeker
	totalSize     int64
	currentPos    int64
}

func (r *ChunkReader) Read(p []byte) (n int, err error) {
	if r.currentPos >= r.totalSize {
		return 0, io.EOF
	}

	if r.currentReader == nil {
		if r.currentIdx >= len(r.chunks) {
			return 0, io.EOF
		}
		// Load next chunk
		chunk := r.chunks[r.currentIdx]

		// Get account
		account, err := r.service.accountService.Get(r.ctx, chunk.AccountID)
		if err != nil {
			return 0, err
		}
		// Ensure token
		if err := r.service.accountService.EnsureTokenValid(r.ctx, account.ID); err != nil {
			return 0, err
		}
		account, _ = r.service.accountService.Get(r.ctx, account.ID)

		client := onedrive.NewClient(account.AccessToken)
		data, err := client.DownloadFile(r.ctx, chunk.RemoteID)
		if err != nil {
			return 0, err
		}

		r.currentReader = bytes.NewReader(data)
		
		// If we seeked into this chunk, we need to adjust the reader
		chunkStartPos := int64(chunk.ChunkIndex) * DefaultChunkSize
		offsetInChunk := r.currentPos - chunkStartPos
		if offsetInChunk > 0 {
			if _, err := r.currentReader.Seek(offsetInChunk, io.SeekStart); err != nil {
				return 0, err
			}
		}
	}

	n, err = r.currentReader.Read(p)
	r.currentPos += int64(n)

	if err == io.EOF {
		r.currentReader = nil
		r.currentIdx++
		if n > 0 {
			return n, nil
		}
		return r.Read(p)
	}
	return n, err
}

func (r *ChunkReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.currentPos + offset
	case io.SeekEnd:
		newPos = r.totalSize + offset
	default:
		return 0, errors.InternalError("invalid whence")
	}

	if newPos < 0 {
		return 0, errors.InternalError("negative position")
	}
	if newPos > r.totalSize {
		newPos = r.totalSize
	}

	r.currentPos = newPos

	// Calculate which chunk contains this position
	chunkIndex := int(newPos / DefaultChunkSize)
	
	// If we are in the same chunk, we might be able to just seek the current reader
	if chunkIndex == r.currentIdx && r.currentReader != nil {
		chunkStartPos := int64(chunkIndex) * DefaultChunkSize
		offsetInChunk := newPos - chunkStartPos
		_, err := r.currentReader.Seek(offsetInChunk, io.SeekStart)
		if err == nil {
			return newPos, nil
		}
		// If seek failed (shouldn't happen with bytes.Reader), fall through to reload
	}

	// Switch to new chunk
	r.currentIdx = chunkIndex
	r.currentReader = nil // Will be loaded on next Read

	return newPos, nil
}

func (r *ChunkReader) Close() error {
	return nil
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
	} else if s.localStorage != nil {
		// Delete from local file storage
		if err := s.localStorage.Delete(bucket, key); err != nil {
			log.Printf("Warning: failed to delete local file: %v", err)
		}
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
