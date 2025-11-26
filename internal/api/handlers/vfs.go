package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/service/vfs"
)

// VFSHandler handles virtual file system requests
type VFSHandler struct {
	vfsService *vfs.Service
}

// NewVFSHandler creates a new VFS handler
func NewVFSHandler(vfsService *vfs.Service) *VFSHandler {
	return &VFSHandler{
		vfsService: vfsService,
	}
}

// UploadFile uploads a file to a virtual path
func (h *VFSHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	path := vars["path"]

	// Ensure path is properly formatted
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Get content type
	mimeType := r.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get content length
	size := r.ContentLength
	if size <= 0 {
		errors.WriteError(w, errors.NewInvalidRequestError("Content-Length header is required"))
		return
	}

	// Upload file
	file, err := h.vfsService.UploadFile(bucket, path, r.Body, size, mimeType)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(file)
}

// Get retrieves a file or lists a directory
func (h *VFSHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	path := vars["path"]

	// Ensure path is properly formatted
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Check if it's a directory listing request (path ends with / or has directory query param)
	isDir := strings.HasSuffix(path, "/") || r.URL.Query().Get("type") == "directory"

	if isDir {
		// List directory
		h.listDirectory(w, r, bucket, path)
	} else {
		// Download file
		h.downloadFile(w, r, bucket, path)
	}
}

// downloadFile downloads a file
func (h *VFSHandler) downloadFile(w http.ResponseWriter, r *http.Request, bucket, path string) {
	data, file, err := h.vfsService.DownloadFile(bucket, path)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	w.Header().Set("ETag", file.ID)
	w.Header().Set("Last-Modified", file.UpdatedAt.Format(http.TimeFormat))

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// listDirectory lists directory contents
func (h *VFSHandler) listDirectory(w http.ResponseWriter, r *http.Request, bucket, path string) {
	// Get query parameters
	recursive := r.URL.Query().Get("recursive") == "true"

	items, err := h.vfsService.ListDirectory(bucket, path, recursive)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	response := map[string]interface{}{
		"path":  path,
		"items": items,
		"total": len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Delete deletes a file or directory
func (h *VFSHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	path := vars["path"]

	// Ensure path is properly formatted
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Check if it's a directory
	isDir := strings.HasSuffix(path, "/") || r.URL.Query().Get("type") == "directory"
	recursive := r.URL.Query().Get("recursive") == "true"

	var err error
	if isDir {
		// Delete directory
		err = h.vfsService.DeleteDirectory(bucket, path, recursive)
	} else {
		// Delete file
		err = h.vfsService.DeleteFile(bucket, path)
	}

	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateDirectory creates a new directory
func (h *VFSHandler) CreateDirectory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Parse request body for path
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteError(w, errors.NewInvalidRequestError("invalid request body"))
		return
	}

	path := req.Path
	if path == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("path is required"))
		return
	}

	// Ensure path is properly formatted
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	dir, err := h.vfsService.CreateDirectory(bucket, path)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dir)
}

// Move moves or renames a file or directory
func (h *VFSHandler) Move(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Parse request body
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteError(w, errors.NewInvalidRequestError("invalid request body"))
		return
	}

	// Validate request
	if req.Source == "" || req.Destination == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("source and destination are required"))
		return
	}

	// Ensure paths are properly formatted
	if !strings.HasPrefix(req.Source, "/") {
		req.Source = "/" + req.Source
	}
	if !strings.HasPrefix(req.Destination, "/") {
		req.Destination = "/" + req.Destination
	}

	// Determine if it's a directory or file
	isDir := strings.HasSuffix(req.Source, "/")

	var result interface{}
	var err error

	if isDir {
		result, err = h.vfsService.MoveDirectory(bucket, req.Source, req.Destination)
	} else {
		result, err = h.vfsService.MoveFile(bucket, req.Source, req.Destination)
	}

	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// Copy copies a file or directory
func (h *VFSHandler) Copy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Parse request body
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteError(w, errors.NewInvalidRequestError("invalid request body"))
		return
	}

	// Validate request
	if req.Source == "" || req.Destination == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("source and destination are required"))
		return
	}

	// Ensure paths are properly formatted
	if !strings.HasPrefix(req.Source, "/") {
		req.Source = "/" + req.Source
	}
	if !strings.HasPrefix(req.Destination, "/") {
		req.Destination = "/" + req.Destination
	}

	// For now, copy is implemented by downloading and re-uploading
	// This is a simple implementation
	isDir := strings.HasSuffix(req.Source, "/")

	if isDir {
		// Directory copy not yet implemented
		errors.WriteError(w, errors.NewInvalidRequestError("directory copy not yet implemented"))
		return
	}

	// Copy file by downloading and re-uploading
	data, file, err := h.vfsService.DownloadFile(bucket, req.Source)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	// Upload to new location
	newFile, err := h.vfsService.UploadFile(bucket, req.Destination, bytes.NewReader(data), file.Size, file.MimeType)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newFile)
}

// Head retrieves file metadata
func (h *VFSHandler) Head(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	path := vars["path"]

	// Ensure path is properly formatted
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	file, err := h.vfsService.GetFile(bucket, path)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	w.Header().Set("ETag", file.ID)
	w.Header().Set("Last-Modified", file.UpdatedAt.Format(http.TimeFormat))

	w.WriteHeader(http.StatusOK)
}
