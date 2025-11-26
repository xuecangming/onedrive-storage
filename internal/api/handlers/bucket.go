package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/service/bucket"
)

// BucketHandler handles bucket-related requests
type BucketHandler struct {
	service *bucket.Service
}

// NewBucketHandler creates a new bucket handler
func NewBucketHandler(service *bucket.Service) *BucketHandler {
	return &BucketHandler{service: service}
}

// List handles GET /buckets
func (h *BucketHandler) List(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.service.List(r.Context())
	if err != nil {
		handleError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"buckets": buckets,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create handles PUT /buckets/{bucket}
func (h *BucketHandler) Create(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.Create(r.Context(), bucketName)
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bucket)
}

// Delete handles DELETE /buckets/{bucket}
func (h *BucketHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	if err := h.service.Delete(r.Context(), bucketName); err != nil {
		handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleError handles application errors
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.HTTPStatus)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": appErr,
		})
		return
	}

	// Generic error
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		},
	})
}
