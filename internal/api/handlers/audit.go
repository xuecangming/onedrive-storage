package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/service/audit"
)

// AuditHandler handles audit requests
type AuditHandler struct {
	auditService *audit.Service
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService *audit.Service) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// StartAudit starts a new audit
func (h *AuditHandler) StartAudit(w http.ResponseWriter, r *http.Request) {
	report, err := h.auditService.StartAudit(r.Context())
	if err != nil {
		errors.WriteError(w, errors.NewConflictError(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(report)
}

// GetStatus retrieves the status of the current or last audit
func (h *AuditHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	report := h.auditService.GetStatus()
	if report == nil {
		errors.WriteError(w, errors.NewNotFoundError("no audit report found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
