package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
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

	// Return success page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>授权成功</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 50px 40px;
            text-align: center;
            max-width: 450px;
        }
        .success-icon {
            width: 80px;
            height: 80px;
            background: linear-gradient(135deg, #28a745 0%%, #20c997 100%%);
            border-radius: 50%%;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 40px;
            margin-bottom: 25px;
            animation: bounceIn 0.6s ease;
        }
        @keyframes bounceIn {
            0%% { transform: scale(0); }
            50%% { transform: scale(1.2); }
            100%% { transform: scale(1); }
        }
        h1 { color: #28a745; font-size: 1.8em; margin-bottom: 15px; }
        .info { color: #666; margin: 20px 0; line-height: 1.8; }
        .info strong { color: #333; }
        .btn {
            display: inline-block;
            padding: 14px 30px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            text-decoration: none;
            border-radius: 10px;
            font-weight: 600;
            margin-top: 20px;
            transition: transform 0.2s;
        }
        .btn:hover { transform: translateY(-2px); }
        .countdown { color: #999; font-size: 0.85em; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="card">
        <div class="success-icon">✓</div>
        <h1>授权成功！</h1>
        <div class="info">
            <p><strong>账号名称:</strong> %s</p>
            <p><strong>邮箱地址:</strong> %s</p>
            <p>OneDrive 已成功连接，现在可以开始使用了！</p>
        </div>
        <a href="/api/v1/oauth/accounts" class="btn">📂 管理账号</a>
        <a href="/" class="btn" style="margin-left: 10px;">🏠 返回主页</a>
        <p class="countdown">页面将在 <span id="timer">5</span> 秒后自动跳转...</p>
    </div>
    <script>
        let seconds = 5;
        const timer = document.getElementById('timer');
        const interval = setInterval(() => {
            seconds--;
            timer.textContent = seconds;
            if (seconds <= 0) {
                clearInterval(interval);
                window.location.href = '/api/v1/oauth/accounts';
            }
        }, 1000);
    </script>
</body>
</html>
`, acc.Name, acc.Email)
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
	fmt.Fprintf(w, setupGuideHTML, redirectURI, redirectURI)
}

const setupGuideHTML = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OneDrive 配置指南</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f7fa;
            line-height: 1.6;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 60px 20px;
            text-align: center;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .header p { opacity: 0.9; font-size: 1.1em; }
        .container { max-width: 800px; margin: 0 auto; padding: 40px 20px; }
        .step {
            background: white;
            border-radius: 16px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.08);
            padding: 30px;
            margin-bottom: 25px;
            position: relative;
        }
        .step-number {
            position: absolute;
            top: -15px;
            left: 30px;
            width: 40px;
            height: 40px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            border-radius: 50%%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            font-size: 1.2em;
        }
        .step h2 { color: #333; margin-bottom: 15px; padding-top: 10px; }
        .step ol { padding-left: 20px; color: #555; }
        .step li { margin: 12px 0; }
        .step a { color: #667eea; }
        code {
            background: #f0f2f5;
            padding: 3px 8px;
            border-radius: 4px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.9em;
        }
        pre {
            background: #2d3748;
            color: #e2e8f0;
            padding: 20px;
            border-radius: 10px;
            overflow-x: auto;
            margin: 15px 0;
        }
        pre code { background: none; color: inherit; }
        .warning {
            background: #fff8e6;
            border-left: 4px solid #f6ad55;
            padding: 20px;
            border-radius: 0 10px 10px 0;
            margin: 20px 0;
        }
        .warning h4 { color: #c05621; margin-bottom: 10px; }
        .tip {
            background: #e6fffa;
            border-left: 4px solid #38b2ac;
            padding: 20px;
            border-radius: 0 10px 10px 0;
            margin: 20px 0;
        }
        .tip h4 { color: #234e52; margin-bottom: 10px; }
        .btn {
            display: inline-block;
            padding: 14px 28px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            text-decoration: none;
            border-radius: 10px;
            font-weight: 600;
            transition: transform 0.2s;
        }
        .btn:hover { transform: translateY(-2px); }
        .btn-secondary {
            background: white;
            color: #667eea;
            border: 2px solid #667eea;
        }
        .actions { text-align: center; margin: 40px 0; }
        .actions .btn { margin: 0 10px; }
        .copy-btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 5px 12px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 0.8em;
            margin-left: 10px;
        }
        .copy-btn:hover { background: #5a67d8; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📘 OneDrive 配置指南</h1>
        <p>按照以下步骤配置您的 Azure AD 应用</p>
    </div>
    
    <div class="container">
        <div class="step">
            <div class="step-number">1</div>
            <h2>🌐 创建 Azure AD 应用</h2>
            <ol>
                <li>访问 <a href="https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" target="_blank">Azure Portal - 应用注册</a></li>
                <li>点击 <strong>"+ 新注册"</strong> 按钮</li>
                <li>填写应用名称，如 <code>OneDrive Storage</code></li>
                <li>选择支持的账户类型（推荐选择 "任何组织目录中的账户和个人 Microsoft 账户"）</li>
                <li>点击 <strong>"注册"</strong> 完成创建</li>
            </ol>
        </div>
        
        <div class="step">
            <div class="step-number">2</div>
            <h2>🔑 获取应用凭据</h2>
            <ol>
                <li>在应用概述页面，复制 <strong>应用程序(客户端) ID</strong> → 这是 Client ID</li>
                <li>复制 <strong>目录(租户) ID</strong> → 这是 Tenant ID</li>
                <li>进入左侧菜单 <strong>"证书和密码"</strong></li>
                <li>点击 <strong>"+ 新客户端密码"</strong></li>
                <li>设置描述和有效期，点击 <strong>"添加"</strong></li>
                <li>⚠️ <strong>立即复制</strong>生成的密码值 → 这是 Client Secret（之后无法再次查看！）</li>
            </ol>
        </div>
        
        <div class="step">
            <div class="step-number">3</div>
            <h2>🔗 配置重定向 URI</h2>
            <ol>
                <li>进入左侧菜单 <strong>"身份验证"</strong></li>
                <li>点击 <strong>"+ 添加平台"</strong> → 选择 <strong>"Web"</strong></li>
                <li>在重定向 URI 中填入：</li>
            </ol>
            <pre><code>%s</code><button class="copy-btn" onclick="copyToClipboard('%s')">📋 复制</button></pre>
            <li>点击 <strong>"配置"</strong> 保存</li>
        </div>
        
        <div class="step">
            <div class="step-number">4</div>
            <h2>📋 配置 API 权限</h2>
            <ol>
                <li>进入左侧菜单 <strong>"API 权限"</strong></li>
                <li>点击 <strong>"+ 添加权限"</strong> → 选择 <strong>"Microsoft Graph"</strong></li>
                <li>选择 <strong>"委托的权限"</strong></li>
                <li>搜索并勾选以下权限：
                    <ul style="margin-top: 10px;">
                        <li><code>Files.ReadWrite.All</code> - 读写所有文件</li>
                        <li><code>offline_access</code> - 保持访问权限（刷新令牌）</li>
                    </ul>
                </li>
                <li>点击 <strong>"添加权限"</strong></li>
                <li>如果您是管理员，点击 <strong>"授予管理员同意"</strong></li>
            </ol>
        </div>
        
        <div class="step">
            <div class="step-number">5</div>
            <h2>✨ 添加账号到系统</h2>
            <p>现在您已经准备好了所有必需的信息，可以添加 OneDrive 账号了！</p>
            <div class="tip">
                <h4>💡 您需要的信息</h4>
                <ul>
                    <li><strong>Client ID</strong> - 应用程序(客户端) ID</li>
                    <li><strong>Client Secret</strong> - 刚才创建的客户端密码</li>
                    <li><strong>Tenant ID</strong> - 目录(租户) ID，或使用 <code>common</code></li>
                </ul>
            </div>
        </div>
        
        <div class="warning">
            <h4>⚠️ 重要提醒</h4>
            <ul>
                <li>Client Secret 请妥善保管，不要分享给他人</li>
                <li>Access Token 有效期约 1 小时，系统会自动刷新</li>
                <li>确保重定向 URI 与 Azure 配置完全一致</li>
                <li>建议设置较长的 Client Secret 有效期（如 24 个月）</li>
            </ul>
        </div>
        
        <div class="actions">
            <a href="/api/v1/oauth/create" class="btn">➕ 立即添加账号</a>
            <a href="/api/v1/oauth/accounts" class="btn btn-secondary">📂 查看账号列表</a>
        </div>
    </div>
    
    <script>
        function copyToClipboard(text) {
            navigator.clipboard.writeText(text).then(() => {
                alert('已复制到剪贴板！');
            });
        }
    </script>
</body>
</html>
`

// CreateAccount handles quick account creation with form data
func (h *OAuthHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show form
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, accountFormHTML)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	html := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OneDrive 账号管理</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container { max-width: 900px; margin: 0 auto; }
        .header {
            text-align: center;
            color: white;
            margin-bottom: 30px;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .header p { opacity: 0.9; }
        .card {
            background: white;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 30px;
            margin-bottom: 20px;
        }
        .add-btn {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 14px 28px;
            border-radius: 10px;
            text-decoration: none;
            font-weight: 600;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .add-btn:hover { transform: translateY(-2px); box-shadow: 0 5px 20px rgba(102,126,234,0.4); }
        .account-list { margin-top: 20px; }
        .account-item {
            display: flex;
            align-items: center;
            padding: 20px;
            border: 1px solid #eee;
            border-radius: 12px;
            margin-bottom: 15px;
            transition: box-shadow 0.2s;
        }
        .account-item:hover { box-shadow: 0 5px 20px rgba(0,0,0,0.1); }
        .account-icon {
            width: 50px;
            height: 50px;
            background: linear-gradient(135deg, #0078d4 0%, #00a4ef 100%);
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 24px;
            margin-right: 15px;
        }
        .account-info { flex: 1; }
        .account-name { font-weight: 600; font-size: 1.1em; color: #333; }
        .account-email { color: #666; font-size: 0.9em; margin-top: 3px; }
        .account-space { color: #888; font-size: 0.85em; margin-top: 5px; }
        .account-status {
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 0.8em;
            font-weight: 500;
        }
        .status-active { background: #d4edda; color: #155724; }
        .status-pending { background: #fff3cd; color: #856404; }
        .status-error { background: #f8d7da; color: #721c24; }
        .account-actions { margin-left: 15px; display: flex; gap: 8px; }
        .action-btn {
            padding: 8px 16px;
            border-radius: 8px;
            border: none;
            cursor: pointer;
            font-size: 0.85em;
            transition: all 0.2s;
        }
        .auth-btn { background: #0078d4; color: white; }
        .auth-btn:hover { background: #006cbd; }
        .sync-btn { background: #28a745; color: white; }
        .sync-btn:hover { background: #218838; }
        .delete-btn { background: #dc3545; color: white; }
        .delete-btn:hover { background: #c82333; }
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #666;
        }
        .empty-state svg { width: 80px; height: 80px; opacity: 0.3; margin-bottom: 20px; }
        .nav-links { margin-top: 20px; text-align: center; }
        .nav-links a { color: white; margin: 0 15px; opacity: 0.8; }
        .nav-links a:hover { opacity: 1; }
        .progress-bar {
            height: 6px;
            background: #e9ecef;
            border-radius: 3px;
            margin-top: 8px;
            overflow: hidden;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea, #764ba2);
            border-radius: 3px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>☁️ OneDrive 账号管理</h1>
            <p>管理您的 OneDrive 存储账号</p>
        </div>
        
        <div class="card">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                <h2>📂 存储账号</h2>
                <a href="/api/v1/oauth/create" class="add-btn">
                    <span>➕</span> 添加账号
                </a>
            </div>
            
            <div class="account-list">`

	if len(accounts) == 0 {
		html += `
                <div class="empty-state">
                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/></svg>
                    <h3>暂无存储账号</h3>
                    <p>点击上方按钮添加您的第一个 OneDrive 账号</p>
                </div>`
	} else {
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

			html += fmt.Sprintf(`
                <div class="account-item">
                    <div class="account-icon">📁</div>
                    <div class="account-info">
                        <div class="account-name">%s</div>
                        <div class="account-email">%s</div>
                        <div class="account-space">%s</div>
                        <div class="progress-bar"><div class="progress-fill" style="width: %.1f%%"></div></div>
                    </div>
                    <span class="account-status %s">%s</span>
                    <div class="account-actions">
                        <button class="action-btn auth-btn" onclick="location.href='/api/v1/oauth/authorize/%s'">🔑 授权</button>
                        <button class="action-btn sync-btn" onclick="syncAccount('%s')">🔄 同步</button>
                        <button class="action-btn delete-btn" onclick="deleteAccount('%s')">🗑️</button>
                    </div>
                </div>`, acc.Name, acc.Email, spaceInfo, usedPercent, statusClass, statusText, acc.ID, acc.ID, acc.ID)
		}
	}

	html += `
            </div>
        </div>
        
        <div class="nav-links">
            <a href="/">← 返回主页</a>
            <a href="/api/v1/oauth/setup">📖 配置指南</a>
        </div>
    </div>
    
    <script>
        async function syncAccount(id) {
            try {
                const resp = await fetch('/api/v1/accounts/' + id + '/sync', { method: 'POST' });
                if (resp.ok) {
                    alert('同步成功！');
                    location.reload();
                } else {
                    alert('同步失败: ' + await resp.text());
                }
            } catch(e) {
                alert('同步失败: ' + e.message);
            }
        }
        
        async function deleteAccount(id) {
            if (!confirm('确定要删除此账号吗？')) return;
            try {
                const resp = await fetch('/api/v1/accounts/' + id, { method: 'DELETE' });
                if (resp.ok) {
                    location.reload();
                } else {
                    alert('删除失败: ' + await resp.text());
                }
            } catch(e) {
                alert('删除失败: ' + e.message);
            }
        }
    </script>
</body>
</html>`

	fmt.Fprint(w, html)
}

const accountFormHTML = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>添加 OneDrive 账号</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
            width: 100%;
            max-width: 500px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo-icon {
            width: 70px;
            height: 70px;
            background: linear-gradient(135deg, #0078d4 0%, #00a4ef 100%);
            border-radius: 18px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 35px;
            margin-bottom: 15px;
        }
        .logo h1 { font-size: 1.5em; color: #333; }
        .logo p { color: #666; font-size: 0.9em; margin-top: 5px; }
        
        .form-group { margin-bottom: 20px; }
        .form-group label {
            display: block;
            font-weight: 600;
            color: #333;
            margin-bottom: 8px;
        }
        .form-group input {
            width: 100%;
            padding: 14px 16px;
            border: 2px solid #e1e5e9;
            border-radius: 10px;
            font-size: 1em;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102,126,234,0.2);
        }
        .form-group .hint {
            font-size: 0.8em;
            color: #888;
            margin-top: 5px;
        }
        .form-group .hint a { color: #667eea; }
        
        .submit-btn {
            width: 100%;
            padding: 16px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 10px;
            font-size: 1.1em;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .submit-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 20px rgba(102,126,234,0.4);
        }
        
        .divider {
            display: flex;
            align-items: center;
            margin: 25px 0;
            color: #999;
            font-size: 0.85em;
        }
        .divider::before, .divider::after {
            content: '';
            flex: 1;
            height: 1px;
            background: #e1e5e9;
        }
        .divider span { padding: 0 15px; }
        
        .help-section {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            margin-top: 20px;
        }
        .help-section h3 { font-size: 0.95em; color: #333; margin-bottom: 12px; }
        .help-steps { font-size: 0.85em; color: #666; line-height: 1.8; }
        .help-steps a { color: #667eea; text-decoration: none; }
        .help-steps a:hover { text-decoration: underline; }
        
        .back-link {
            display: block;
            text-align: center;
            margin-top: 20px;
            color: #667eea;
            text-decoration: none;
        }
        .back-link:hover { text-decoration: underline; }
        
        .tenant-presets {
            display: flex;
            gap: 8px;
            margin-top: 8px;
        }
        .preset-btn {
            padding: 6px 12px;
            background: #f0f0f0;
            border: none;
            border-radius: 6px;
            font-size: 0.8em;
            cursor: pointer;
            transition: background 0.2s;
        }
        .preset-btn:hover { background: #e0e0e0; }
    </style>
</head>
<body>
    <div class="card">
        <div class="logo">
            <div class="logo-icon">☁️</div>
            <h1>添加 OneDrive 账号</h1>
            <p>连接您的 Microsoft 365 存储空间</p>
        </div>
        
        <form method="POST" action="/api/v1/oauth/create">
            <div class="form-group">
                <label>📛 账号名称</label>
                <input type="text" name="name" placeholder="如：主账号、备份账号" required>
                <div class="hint">用于识别此存储账号的名称</div>
            </div>
            
            <div class="form-group">
                <label>📧 邮箱地址</label>
                <input type="email" name="email" placeholder="your-email@outlook.com" required>
                <div class="hint">OneDrive 账号对应的 Microsoft 邮箱</div>
            </div>
            
            <div class="form-group">
                <label>🔑 Client ID (客户端 ID)</label>
                <input type="text" name="client_id" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" required>
                <div class="hint">Azure AD 应用的客户端 ID，<a href="/api/v1/oauth/setup" target="_blank">查看获取方式</a></div>
            </div>
            
            <div class="form-group">
                <label>🔐 Client Secret (客户端密码)</label>
                <input type="password" name="client_secret" placeholder="您的客户端密码" required>
                <div class="hint">Azure AD 应用创建的客户端密码</div>
            </div>
            
            <div class="form-group">
                <label>🏢 Tenant ID (租户 ID)</label>
                <input type="text" name="tenant_id" id="tenant_id" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" required>
                <div class="hint">Azure AD 的租户 ID</div>
                <div class="tenant-presets">
                    <button type="button" class="preset-btn" onclick="document.getElementById('tenant_id').value='common'">通用 (common)</button>
                    <button type="button" class="preset-btn" onclick="document.getElementById('tenant_id').value='organizations'">组织 (organizations)</button>
                    <button type="button" class="preset-btn" onclick="document.getElementById('tenant_id').value='consumers'">个人 (consumers)</button>
                </div>
            </div>
            
            <button type="submit" class="submit-btn">🚀 创建并授权</button>
        </form>
        
        <div class="divider"><span>需要帮助?</span></div>
        
        <div class="help-section">
            <h3>📖 快速开始</h3>
            <div class="help-steps">
                1. 前往 <a href="https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" target="_blank">Azure Portal</a> 创建应用<br>
                2. 获取 Client ID 和 Tenant ID<br>
                3. 创建 Client Secret<br>
                4. 设置重定向 URI<br>
                <a href="/api/v1/oauth/setup" target="_blank">📘 查看详细配置指南 →</a>
            </div>
        </div>
        
        <a href="/api/v1/oauth/accounts" class="back-link">← 返回账号列表</a>
    </div>
</body>
</html>
`
