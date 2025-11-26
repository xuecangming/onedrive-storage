package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WebHandler handles static file serving for the web application
type WebHandler struct {
	staticDir string
}

// NewWebHandler creates a new web handler
func NewWebHandler(staticDir string) *WebHandler {
	return &WebHandler{
		staticDir: staticDir,
	}
}

// ServeStatic serves static files from the static directory
func (h *WebHandler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Get the path after /static/
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	// Prevent directory traversal
	path = filepath.Clean(path)
	if strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(h.staticDir, path)

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Don't serve directories
	if info.IsDir() {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Set content type based on extension
	ext := filepath.Ext(filePath)
	contentType := getContentType(ext)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeFile(w, r, filePath)
}

// ServeIndex serves the index.html file
func (h *WebHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(h.staticDir, "index.html")

	// Check if index.html exists
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, indexPath)
}

// getContentType returns the content type for a file extension
func getContentType(ext string) string {
	contentTypes := map[string]string{
		".html":  "text/html; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".js":    "application/javascript; charset=utf-8",
		".json":  "application/json",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".eot":   "application/vnd.ms-fontobject",
	}
	return contentTypes[ext]
}
