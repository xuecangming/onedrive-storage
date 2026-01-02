package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/onedrive"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

// OAuthHandler handles OAuth authentication flow
type OAuthHandler struct {
	accountService *account.Service
	baseURL        string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(accountService *account.Service, baseURL string) *OAuthHandler {
	return &OAuthHandler{
		accountService: accountService,
		baseURL:        baseURL,
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

	// Redirect to root (frontend)
	http.Redirect(w, r, "/", http.StatusFound)
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
