package errors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := NewAppError(ErrInvalidBucket, "test message", http.StatusBadRequest)
	
	expected := "[INVALID_BUCKET] test message"
	if err.Error() != expected {
		t.Errorf("AppError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestAppError_WithDetails(t *testing.T) {
	err := NewAppError(ErrBucketNotFound, "not found", http.StatusNotFound).
		WithDetails("bucket", "my-bucket").
		WithDetails("region", "us-east-1")

	if err.Details["bucket"] != "my-bucket" {
		t.Errorf("Details[bucket] = %v, want 'my-bucket'", err.Details["bucket"])
	}
	if err.Details["region"] != "us-east-1" {
		t.Errorf("Details[region] = %v, want 'us-east-1'", err.Details["region"])
	}
}

func TestInvalidBucket(t *testing.T) {
	err := InvalidBucket("bad-bucket")

	if err.Code != ErrInvalidBucket {
		t.Errorf("Code = %v, want %v", err.Code, ErrInvalidBucket)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusBadRequest)
	}
	if err.Details["bucket"] != "bad-bucket" {
		t.Errorf("Details[bucket] = %v, want 'bad-bucket'", err.Details["bucket"])
	}
}

func TestBucketNotFound(t *testing.T) {
	err := BucketNotFound("missing-bucket")

	if err.Code != ErrBucketNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrBucketNotFound)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusNotFound)
	}
}

func TestObjectNotFound(t *testing.T) {
	err := ObjectNotFound("my-bucket", "my-key")

	if err.Code != ErrObjectNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrObjectNotFound)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusNotFound)
	}
	if err.Details["bucket"] != "my-bucket" {
		t.Errorf("Details[bucket] = %v, want 'my-bucket'", err.Details["bucket"])
	}
	if err.Details["key"] != "my-key" {
		t.Errorf("Details[key] = %v, want 'my-key'", err.Details["key"])
	}
}

func TestBucketExists(t *testing.T) {
	err := BucketExists("existing-bucket")

	if err.Code != ErrBucketExists {
		t.Errorf("Code = %v, want %v", err.Code, ErrBucketExists)
	}
	if err.HTTPStatus != http.StatusConflict {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusConflict)
	}
}

func TestFileTooLarge(t *testing.T) {
	err := FileTooLarge(1000, 500)

	if err.Code != ErrFileTooLarge {
		t.Errorf("Code = %v, want %v", err.Code, ErrFileTooLarge)
	}
	if err.HTTPStatus != http.StatusRequestEntityTooLarge {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusRequestEntityTooLarge)
	}
	if err.Details["size"] != int64(1000) {
		t.Errorf("Details[size] = %v, want 1000", err.Details["size"])
	}
	if err.Details["max_size"] != int64(500) {
		t.Errorf("Details[max_size] = %v, want 500", err.Details["max_size"])
	}
}

func TestInternalError(t *testing.T) {
	err := InternalError("something went wrong")

	if err.Code != ErrInternal {
		t.Errorf("Code = %v, want %v", err.Code, ErrInternal)
	}
	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusInternalServerError)
	}
	if err.Message != "something went wrong" {
		t.Errorf("Message = %v, want 'something went wrong'", err.Message)
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "AppError",
			err:            BucketNotFound("test-bucket"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "BUCKET_NOT_FOUND",
		},
		{
			name:           "generic error",
			err:            &testError{msg: "generic error"},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("status code = %v, want %v", w.Code, tt.expectedStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedCode) {
				t.Errorf("body = %v, should contain %v", body, tt.expectedCode)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
