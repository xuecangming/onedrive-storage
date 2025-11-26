package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xuecangming/onedrive-storage/internal/core/loadbalancer"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

// SpaceHandler handles space management requests
type SpaceHandler struct {
	accountService *account.Service
	balancer       *loadbalancer.Balancer
}

// NewSpaceHandler creates a new space handler
func NewSpaceHandler(accountService *account.Service) *SpaceHandler {
	return &SpaceHandler{
		accountService: accountService,
		balancer:       loadbalancer.NewBalancer(loadbalancer.StrategyLeastUsed),
	}
}

// Overview handles GET /space
func (h *SpaceHandler) Overview(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.accountService.List(r.Context())
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Calculate statistics
	stats := h.balancer.GetUsageStats(accounts)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ListAccounts handles GET /space/accounts
func (h *SpaceHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.accountService.List(r.Context())
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Format account info for space view
	var accountsInfo []map[string]interface{}
	for _, acc := range accounts {
		availableSpace := acc.TotalSpace - acc.UsedSpace
		usagePercent := 0.0
		if acc.TotalSpace > 0 {
			usagePercent = float64(acc.UsedSpace) / float64(acc.TotalSpace) * 100
		}

		accountsInfo = append(accountsInfo, map[string]interface{}{
			"id":              acc.ID,
			"name":            acc.Name,
			"email":           acc.Email,
			"status":          acc.Status,
			"total_space":     acc.TotalSpace,
			"used_space":      acc.UsedSpace,
			"available_space": availableSpace,
			"usage_percent":   usagePercent,
			"priority":        acc.Priority,
			"last_sync":       acc.LastSync,
		})
	}

	response := map[string]interface{}{
		"accounts": accountsInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AccountDetail handles GET /space/accounts/{id}
func (h *SpaceHandler) AccountDetail(w http.ResponseWriter, r *http.Request) {
	// Reuse account handler
	accountHandler := NewAccountHandler(h.accountService)
	accountHandler.Get(w, r)
}

// SyncAccount handles POST /space/accounts/{id}/sync
func (h *SpaceHandler) SyncAccount(w http.ResponseWriter, r *http.Request) {
	// Reuse account handler
	accountHandler := NewAccountHandler(h.accountService)
	accountHandler.SyncSpace(w, r)
}
