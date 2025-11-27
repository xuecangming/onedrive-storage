package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/service/vfs"
)

// EnhancedVFSHandler handles enhanced VFS requests (starred, trash, recent, search)
type EnhancedVFSHandler struct {
	enhancedService *vfs.EnhancedService
}

// NewEnhancedVFSHandler creates a new enhanced VFS handler
func NewEnhancedVFSHandler(enhancedService *vfs.EnhancedService) *EnhancedVFSHandler {
	return &EnhancedVFSHandler{
		enhancedService: enhancedService,
	}
}

// ==================== Starred Files ====================

// StarFile stars a file
func (h *EnhancedVFSHandler) StarFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	var req struct {
		FileID   string `json:"file_id"`
		FilePath string `json:"file_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteError(w, errors.NewInvalidRequestError("invalid request body"))
		return
	}

	if req.FileID == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("file_id is required"))
		return
	}

	if err := h.enhancedService.StarFile(bucket, req.FileID, req.FilePath); err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "file starred successfully",
		"file_id": req.FileID,
	})
}

// UnstarFile unstars a file
func (h *EnhancedVFSHandler) UnstarFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	fileID := vars["file_id"]

	if fileID == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("file_id is required"))
		return
	}

	if err := h.enhancedService.UnstarFile(bucket, fileID); err != nil {
		errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStarredFiles returns all starred files
func (h *EnhancedVFSHandler) GetStarredFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	items, err := h.enhancedService.GetStarredFiles(bucket)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	response := map[string]interface{}{
		"items": items,
		"total": len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ==================== Trash ====================

// GetTrashItems returns all items in trash
func (h *EnhancedVFSHandler) GetTrashItems(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	items, err := h.enhancedService.GetTrashItems(bucket)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	response := map[string]interface{}{
		"items": items,
		"total": len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RestoreFromTrash restores an item from trash
func (h *EnhancedVFSHandler) RestoreFromTrash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trashID := vars["trash_id"]

	if trashID == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("trash_id is required"))
		return
	}

	if err := h.enhancedService.RestoreFromTrash(trashID); err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "item restored successfully",
	})
}

// DeleteFromTrash permanently deletes an item from trash
func (h *EnhancedVFSHandler) DeleteFromTrash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trashID := vars["trash_id"]

	if trashID == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("trash_id is required"))
		return
	}

	if err := h.enhancedService.DeleteFromTrash(trashID); err != nil {
		errors.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EmptyTrash empties the trash
func (h *EnhancedVFSHandler) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	count, err := h.enhancedService.EmptyTrash(bucket)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "trash emptied successfully",
		"deleted_count": count,
	})
}

// ==================== Recent Files ====================

// GetRecentFiles returns recently accessed files
func (h *EnhancedVFSHandler) GetRecentFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Parse limit parameter
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	items, err := h.enhancedService.GetRecentFiles(bucket, limit)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	response := map[string]interface{}{
		"items": items,
		"total": len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ==================== Search ====================

// Search searches for files and directories
func (h *EnhancedVFSHandler) Search(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Parse query parameters
	query := r.URL.Query().Get("q")
	if query == "" {
		errors.WriteError(w, errors.NewInvalidRequestError("query parameter 'q' is required"))
		return
	}

	// Parse limit parameter
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Check for type filter
	fileType := r.URL.Query().Get("type")
	
	var results interface{}
	var err error

	if fileType != "" {
		results, err = h.enhancedService.SearchByType(bucket, fileType, limit)
	} else {
		results, err = h.enhancedService.Search(bucket, query, limit)
	}

	if err != nil {
		errors.WriteError(w, err)
		return
	}

	response := map[string]interface{}{
		"query":   query,
		"results": results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFilesByDateRange returns files within a date range
func (h *EnhancedVFSHandler) GetFilesByDateRange(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]

	// Parse date parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			errors.WriteError(w, errors.NewInvalidRequestError("invalid 'from' date format, use RFC3339"))
			return
		}
	} else {
		// Default to 7 days ago
		from = time.Now().AddDate(0, 0, -7)
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			errors.WriteError(w, errors.NewInvalidRequestError("invalid 'to' date format, use RFC3339"))
			return
		}
	} else {
		to = time.Now()
	}

	// Parse limit parameter
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	items, err := h.enhancedService.GetFilesByDateRange(bucket, from, to, limit)
	if err != nil {
		errors.WriteError(w, err)
		return
	}

	response := map[string]interface{}{
		"from":  from,
		"to":    to,
		"items": items,
		"total": len(items),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
