package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

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
func (h *ObjectHandler) Download(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	key := vars["key"]

	obj, data, err := h.service.Download(r.Context(), bucketName, key)
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", obj.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(obj.Size, 10))
	if obj.ETag != "" {
		w.Header().Set("ETag", obj.ETag)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Head handles HEAD /objects/{bucket}/{key}
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
