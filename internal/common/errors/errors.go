package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application error codes
type ErrorCode string

const (
	// 400 errors
	ErrInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrInvalidBucket  ErrorCode = "INVALID_BUCKET"
	ErrInvalidKey     ErrorCode = "INVALID_KEY"
	ErrInvalidPath    ErrorCode = "INVALID_PATH"

	// 404 errors
	ErrBucketNotFound ErrorCode = "BUCKET_NOT_FOUND"
	ErrObjectNotFound ErrorCode = "OBJECT_NOT_FOUND"
	ErrPathNotFound   ErrorCode = "PATH_NOT_FOUND"

	// 409 errors
	ErrBucketExists   ErrorCode = "BUCKET_EXISTS"
	ErrObjectExists   ErrorCode = "OBJECT_EXISTS"
	ErrPathExists     ErrorCode = "PATH_EXISTS"
	ErrBucketNotEmpty ErrorCode = "BUCKET_NOT_EMPTY"
	ErrDirNotEmpty    ErrorCode = "DIR_NOT_EMPTY"

	// 413, 507 errors
	ErrFileTooLarge  ErrorCode = "FILE_TOO_LARGE"
	ErrStorageFull   ErrorCode = "STORAGE_FULL"

	// 500 errors
	ErrInternal        ErrorCode = "INTERNAL_ERROR"
	ErrUpstreamError   ErrorCode = "UPSTREAM_ERROR"
	ErrServiceUnavail  ErrorCode = "SERVICE_UNAVAIL"
)

// AppError represents an application error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	e.Details[key] = value
	return e
}

// Common error constructors
func InvalidRequest(message string) *AppError {
	return NewAppError(ErrInvalidRequest, message, http.StatusBadRequest)
}

func InvalidBucket(bucketName string) *AppError {
	return NewAppError(ErrInvalidBucket, "Invalid bucket name format", http.StatusBadRequest).
		WithDetails("bucket", bucketName)
}

func InvalidKey(key string) *AppError {
	return NewAppError(ErrInvalidKey, "Invalid object key format", http.StatusBadRequest).
		WithDetails("key", key)
}

func InvalidPath(path string) *AppError {
	return NewAppError(ErrInvalidPath, "Invalid path format", http.StatusBadRequest).
		WithDetails("path", path)
}

func BucketNotFound(bucketName string) *AppError {
	return NewAppError(ErrBucketNotFound, "Bucket not found", http.StatusNotFound).
		WithDetails("bucket", bucketName)
}

func ObjectNotFound(bucket, key string) *AppError {
	return NewAppError(ErrObjectNotFound, "Object not found", http.StatusNotFound).
		WithDetails("bucket", bucket).
		WithDetails("key", key)
}

func PathNotFound(path string) *AppError {
	return NewAppError(ErrPathNotFound, "Path not found", http.StatusNotFound).
		WithDetails("path", path)
}

func BucketExists(bucketName string) *AppError {
	return NewAppError(ErrBucketExists, "Bucket already exists", http.StatusConflict).
		WithDetails("bucket", bucketName)
}

func ObjectExists(bucket, key string) *AppError {
	return NewAppError(ErrObjectExists, "Object already exists", http.StatusConflict).
		WithDetails("bucket", bucket).
		WithDetails("key", key)
}

func BucketNotEmpty(bucketName string) *AppError {
	return NewAppError(ErrBucketNotEmpty, "Bucket is not empty", http.StatusConflict).
		WithDetails("bucket", bucketName)
}

func FileTooLarge(size, maxSize int64) *AppError {
	return NewAppError(ErrFileTooLarge, "File exceeds size limit", http.StatusRequestEntityTooLarge).
		WithDetails("size", size).
		WithDetails("max_size", maxSize)
}

func StorageFull() *AppError {
	return NewAppError(ErrStorageFull, "Insufficient storage space", http.StatusInsufficientStorage)
}

func InternalError(message string) *AppError {
	return NewAppError(ErrInternal, message, http.StatusInternalServerError)
}

func UpstreamError(message string) *AppError {
	return NewAppError(ErrUpstreamError, message, http.StatusBadGateway)
}
