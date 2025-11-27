package handlers

import (
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

	// Read the entire request body
	data, err := io.ReadAll(r.Body)
	if err != nil {
		handleError(w, r, err)
		return
	}
	defer r.Body.Close()

	obj, err := h.service.Upload(r.Context(), bucketName, key, data, contentType)
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

	obj, data, err := h.service.Download(r.Context(), bucketName, key)
	if err != nil {
		handleError(w, r, err)
		return
	}

	totalSize := int64(len(data))
	
	// Set common headers
	w.Header().Set("Content-Type", obj.MimeType)
	if obj.ETag != "" {
		w.Header().Set("ETag", obj.ETag)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Check for Range header
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// No range requested, return full content
		w.Header().Set("Content-Length", strconv.FormatInt(totalSize, 10))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// Parse Range header
	start, end, err := parseRangeHeader(rangeHeader, totalSize)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		http.Error(w, "Requested Range Not Satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Calculate content length for partial content
	contentLength := end - start + 1

	// Set headers for partial content
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	w.WriteHeader(http.StatusPartialContent)

	// Write the requested range
	w.Write(data[start : end+1])
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
