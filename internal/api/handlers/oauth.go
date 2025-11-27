package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/api/templates"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/onedrive"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

// OAuthHandler handles OAuth authentication flow
type OAuthHandler struct {
	accountService *account.Service
	baseURL        string
	tmpl           *templates.Manager
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(accountService *account.Service, baseURL string) *OAuthHandler {
	return &OAuthHandler{
		accountService: accountService,
		baseURL:        baseURL,
		tmpl:           templates.GetManager(),
	}
}

// getRedirectURI returns the OAuth redirect URI, using baseURL if set, otherwise from request
func (h *OAuthHandler) getRedirectURI(r *http.Request) string {
	if h.baseURL != "" {
		return h.baseURL + "/api/v1/oauth/callback"
	}
	
	// Determine scheme
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header (common for reverse proxies)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	
	// Get host from request
	host := r.Host
	// Check X-Forwarded-Host header
	if fwdHost := r.Header.Get("X-Forwarded-Host"); fwdHost != "" {
		host = fwdHost
	}
	
	return fmt.Sprintf("%s://%s/api/v1/oauth/callback", scheme, host)
}

// Authorize handles GET /oauth/authorize/{id}
// Redirects user to Microsoft login page
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get account
	acc, err := h.accountService.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Get dynamic redirect URI from request
	redirectURI := h.getRedirectURI(r)

	// Create auth client
	authConfig := onedrive.AuthConfig{
		ClientID:     acc.ClientID,
		ClientSecret: acc.ClientSecret,
		TenantID:     acc.TenantID,
		RedirectURI:  redirectURI,
	}
	auth := onedrive.NewAuth(authConfig)

	// Generate authorization URL with account ID as state
	authURL := auth.GetAuthorizationURL(id)

	// Redirect to Microsoft login
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Callback handles GET /oauth/callback
// Receives authorization code from Microsoft and exchanges for tokens
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state") // This is the account ID
	errorCode := r.URL.Query().Get("error")

	if errorCode != "" {
		errorDesc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("OAuth error: %s - %s", errorCode, errorDesc), http.StatusBadRequest)
		return
	}

	if code == "" || state == "" {
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}

	// Get account by ID (state)
	acc, err := h.accountService.Get(r.Context(), state)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Get dynamic redirect URI from request
	redirectURI := h.getRedirectURI(r)

	// Create auth client
	authConfig := onedrive.AuthConfig{
		ClientID:     acc.ClientID,
		ClientSecret: acc.ClientSecret,
		TenantID:     acc.TenantID,
		RedirectURI:  redirectURI,
	}
	auth := onedrive.NewAuth(authConfig)

	// Exchange code for tokens
	tokenResp, err := auth.ExchangeCode(r.Context(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
		return
	}

	// Update account with tokens
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	acc.AccessToken = tokenResp.AccessToken
	acc.RefreshToken = tokenResp.RefreshToken
	acc.TokenExpires = expiresAt
	acc.Status = "active"

	if err := h.accountService.Update(r.Context(), acc); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update account: %v", err), http.StatusInternalServerError)
		return
	}

	// Try to sync space info
	_ = h.accountService.SyncSpaceInfo(r.Context(), acc.ID)

	// Return success page using template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := templates.SuccessData{
		Name:  acc.Name,
		Email: acc.Email,
	}
	if err := h.tmpl.Render(w, "success.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// TokenStatus handles GET /oauth/status/{id}
// Returns the token status of an account
func (h *OAuthHandler) TokenStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	acc, err := h.accountService.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	status := map[string]interface{}{
		"id":            acc.ID,
		"name":          acc.Name,
		"email":         acc.Email,
		"status":        acc.Status,
		"has_token":     acc.AccessToken != "",
		"token_expires": acc.TokenExpires,
		"is_expired":    time.Now().After(acc.TokenExpires),
		"total_space":   acc.TotalSpace,
		"used_space":    acc.UsedSpace,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// SetupGuide handles GET /oauth/setup
// Returns HTML guide for setting up OneDrive accounts
func (h *OAuthHandler) SetupGuide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Get dynamic redirect URI from request
	redirectURI := h.getRedirectURI(r)
	data := templates.SetupGuideData{
		RedirectURI: redirectURI,
	}
	if err := h.tmpl.Render(w, "setup_guide.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// CreateAccount handles quick account creation with form data
func (h *OAuthHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show form using template
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.tmpl.Render(w, "account_form.html", nil); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// Handle POST - create account and redirect to authorize
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	acc := &types.StorageAccount{
		Name:         r.FormValue("name"),
		Email:        r.FormValue("email"),
		ClientID:     r.FormValue("client_id"),
		ClientSecret: r.FormValue("client_secret"),
		TenantID:     r.FormValue("tenant_id"),
		Status:       "pending",
		Priority:     10,
	}

	if err := h.accountService.Create(r.Context(), acc); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create account: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to authorization (use StatusSeeOther to convert POST to GET)
	http.Redirect(w, r, fmt.Sprintf("/api/v1/oauth/authorize/%s", acc.ID), http.StatusSeeOther)
}

// AccountList shows all accounts with management UI
func (h *OAuthHandler) AccountList(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.accountService.List(r.Context())
	if err != nil {
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	// Convert accounts to view data
	var viewData []templates.AccountViewData
	for _, acc := range accounts {
		statusClass := "status-pending"
		statusText := "待授权"
		if acc.Status == "active" {
			statusClass = "status-active"
			statusText = "已激活"
		} else if acc.Status == "error" {
			statusClass = "status-error"
			statusText = "错误"
		}

		usedPercent := 0.0
		spaceInfo := "未同步"
		if acc.TotalSpace > 0 {
			usedPercent = float64(acc.UsedSpace) / float64(acc.TotalSpace) * 100
			usedGB := float64(acc.UsedSpace) / (1024 * 1024 * 1024)
			totalGB := float64(acc.TotalSpace) / (1024 * 1024 * 1024)
			spaceInfo = fmt.Sprintf("%.1f GB / %.1f GB (%.1f%%)", usedGB, totalGB, usedPercent)
		}

		viewData = append(viewData, templates.AccountViewData{
			ID:          acc.ID,
			Name:        acc.Name,
			Email:       acc.Email,
			SpaceInfo:   spaceInfo,
			UsedPercent: usedPercent,
			StatusClass: statusClass,
			StatusText:  statusText,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := templates.AccountListData{Accounts: viewData}
	if err := h.tmpl.Render(w, "account_list.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
