package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

// AccountHandler handles account management requests
type AccountHandler struct {
	service *account.Service
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(service *account.Service) *AccountHandler {
	return &AccountHandler{service: service}
}

// List handles GET /accounts
func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.service.List(r.Context())
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Hide sensitive information
	for _, acc := range accounts {
		acc.ClientSecret = ""
		acc.RefreshToken = ""
		acc.AccessToken = ""
	}

	response := map[string]interface{}{
		"accounts": accounts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get handles GET /accounts/{id}
func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	account, err := h.service.Get(r.Context(), id)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Hide sensitive information
	account.ClientSecret = ""
	account.RefreshToken = ""
	account.AccessToken = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// Create handles POST /accounts
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var account types.StorageAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Create(r.Context(), &account); err != nil {
		handleError(w, r, err)
		return
	}

	// Hide sensitive information
	account.ClientSecret = ""
	account.RefreshToken = ""
	account.AccessToken = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
}

// Update handles PUT /accounts/{id}
func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var account types.StorageAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	account.ID = id

	if err := h.service.Update(r.Context(), &account); err != nil {
		handleError(w, r, err)
		return
	}

	// Hide sensitive information
	account.ClientSecret = ""
	account.RefreshToken = ""
	account.AccessToken = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// Delete handles DELETE /accounts/{id}
func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.Delete(r.Context(), id); err != nil {
		handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RefreshToken handles POST /accounts/{id}/refresh
func (h *AccountHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.RefreshToken(r.Context(), id); err != nil {
		handleError(w, r, err)
		return
	}

	response := map[string]string{
		"message": "Token refreshed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncSpace handles POST /accounts/{id}/sync
func (h *AccountHandler) SyncSpace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.SyncSpaceInfo(r.Context(), id); err != nil {
		handleError(w, r, err)
		return
	}

	// Get updated account info
	account, err := h.service.Get(r.Context(), id)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Hide sensitive information
	account.ClientSecret = ""
	account.RefreshToken = ""
	account.AccessToken = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}
