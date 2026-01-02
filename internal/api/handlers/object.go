package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/service/object"
)

// ObjectHandler handles object-related requests
type ObjectHandler struct {
	service *object.Service
}

// NewObjectHandler creates a new object handler
func NewObjectHandler(service *object.Service) *ObjectHandler {
	return &ObjectHandler{service: service}
}

// List handles GET /objects/{bucket}
func (h *ObjectHandler) List(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	prefix := r.URL.Query().Get("prefix")
	marker := r.URL.Query().Get("marker")
	maxKeys := 1000
	if mk := r.URL.Query().Get("max_keys"); mk != "" {
		if parsed, err := strconv.Atoi(mk); err == nil && parsed > 0 && parsed <= 1000 {
			maxKeys = parsed
		}
	}

	objects, nextMarker, isTruncated, err := h.service.List(r.Context(), bucketName, prefix, marker, maxKeys)
	if err != nil {
		handleError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"bucket":       bucketName,
		"prefix":       prefix,
		"objects":      objects,
		"is_truncated": isTruncated,
		"next_marker":  nextMarker,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Upload handles PUT /objects/{bucket}/{key}
func (h *ObjectHandler) Upload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	var reader io.Reader = r.Body
	size := r.ContentLength

	// If Content-Length is missing, we must read the body to know the size
	if size <= 0 {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, err)
			return
		}
		size = int64(len(data))
		reader = bytes.NewReader(data)
	}
	defer r.Body.Close()

	obj, err := h.service.Upload(r.Context(), bucketName, key, reader, size, contentType)
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(obj)
}

// Download handles GET /objects/{bucket}/{key}
// Supports HTTP Range requests for streaming and resumable downloads
func (h *ObjectHandler) Download(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]

	obj, reader, err := h.service.Download(r.Context(), bucketName, key)
	if err != nil {
		handleError(w, r, err)
		return
	}
	defer reader.Close()

	// Use http.ServeContent to handle Range requests automatically
	// It requires an io.ReadSeeker, which our reader now implements
	http.ServeContent(w, r, obj.Key, obj.UpdatedAt, reader)
}

// parseRangeHeader parses the Range header and returns start and end byte positions
// Supports formats: "bytes=0-499", "bytes=500-999", "bytes=500-", "bytes=-500"
func parseRangeHeader(rangeHeader string, totalSize int64) (start, end int64, err error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	
	// Handle multiple ranges - we only support single range for now
	if strings.Contains(rangeSpec, ",") {
		return 0, 0, fmt.Errorf("multiple ranges not supported")
	}

	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	// Parse start position
	if parts[0] == "" {
		// Format: bytes=-500 (last 500 bytes)
		suffixLength, parseErr := strconv.ParseInt(parts[1], 10, 64)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("invalid range format")
		}
		start = totalSize - suffixLength
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
	} else {
		// Format: bytes=500-999 or bytes=500-
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range format")
		}

		if parts[1] == "" {
			// Format: bytes=500- (from byte 500 to end)
			end = totalSize - 1
		} else {
			// Format: bytes=500-999
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid range format")
			}
		}
	}

	// Validate start position - must be within file
	if start < 0 || start >= totalSize {
		return 0, 0, fmt.Errorf("range not satisfiable")
	}

	// Adjust end if it exceeds file size (as per RFC 7233)
	if end >= totalSize {
		end = totalSize - 1
	}

	// Validate that start <= end
	if end < start {
		return 0, 0, fmt.Errorf("range not satisfiable")
	}

	return start, end, nil
}

// Head handles HEAD /objects/{bucket}/{key}
// Returns object metadata including Accept-Ranges header for streaming support
func (h *ObjectHandler) Head(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]

	obj, err := h.service.GetMetadata(r.Context(), bucketName, key)
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", obj.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(obj.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	if obj.ETag != "" {
		w.Header().Set("ETag", obj.ETag)
	}
	w.WriteHeader(http.StatusOK)
}

// Delete handles DELETE /objects/{bucket}/{key}
func (h *ObjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]

	if err := h.service.Delete(r.Context(), bucketName, key); err != nil {
		handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
