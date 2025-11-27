package api

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xuecangming/onedrive-storage/internal/api/handlers"
	"github.com/xuecangming/onedrive-storage/internal/api/middleware"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/repository"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
	"github.com/xuecangming/onedrive-storage/internal/service/bucket"
	"github.com/xuecangming/onedrive-storage/internal/service/object"
	"github.com/xuecangming/onedrive-storage/internal/service/vfs"
)

// Server represents the HTTP server
type Server struct {
	config             *types.Config
	db                 *sql.DB
	router             *mux.Router
	bucketHandler      *handlers.BucketHandler
	objectHandler      *handlers.ObjectHandler
	accountHandler     *handlers.AccountHandler
	spaceHandler       *handlers.SpaceHandler
	healthHandler      *handlers.HealthHandler
	vfsHandler         *handlers.VFSHandler
	oauthHandler       *handlers.OAuthHandler
	enhancedVFSHandler *handlers.EnhancedVFSHandler
}

// NewServer creates a new HTTP server
func NewServer(config *types.Config, db *sql.DB) *Server {
	// Create repositories
	bucketRepo := repository.NewBucketRepository(db)
	objectRepo := repository.NewObjectRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	vfsRepo := repository.NewVFSRepository(db)
	enhancedVFSRepo := repository.NewEnhancedVFSRepository(db)

	// Create services
	bucketService := bucket.NewService(bucketRepo)
	accountService := account.NewService(accountRepo)
	// Use OneDrive integration for real storage
	objectService := object.NewServiceWithOneDrive(objectRepo, bucketRepo, accountService)
	vfsService := vfs.NewService(vfsRepo, objectService, bucketRepo)
	enhancedVFSService := vfs.NewEnhancedService(enhancedVFSRepo, vfsRepo, bucketRepo)

	// Create handlers
	bucketHandler := handlers.NewBucketHandler(bucketService)
	objectHandler := handlers.NewObjectHandler(objectService)
	accountHandler := handlers.NewAccountHandler(accountService)
	spaceHandler := handlers.NewSpaceHandler(accountService)
	healthHandler := handlers.NewHealthHandler(db)
	vfsHandler := handlers.NewVFSHandler(vfsService)
	enhancedVFSHandler := handlers.NewEnhancedVFSHandler(enhancedVFSService)

	// Create OAuth handler (redirect URI will be determined dynamically from request)
	oauthHandler := handlers.NewOAuthHandler(accountService, config.Server.BaseURL)

	server := &Server{
		config:             config,
		db:                 db,
		router:             mux.NewRouter(),
		bucketHandler:      bucketHandler,
		objectHandler:      objectHandler,
		accountHandler:     accountHandler,
		spaceHandler:       spaceHandler,
		healthHandler:      healthHandler,
		vfsHandler:         vfsHandler,
		oauthHandler:       oauthHandler,
		enhancedVFSHandler: enhancedVFSHandler,
	}

	server.setupRoutes()

	return server
}

// Router returns the HTTP router
func (s *Server) Router() http.Handler {
	return s.router
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Apply global middleware
	s.router.Use(middleware.CORSMiddleware)
	s.router.Use(middleware.LoggingMiddleware)
	s.router.Use(middleware.RecoveryMiddleware)

	// API v1 routes
	api := s.router.PathPrefix(s.config.Server.APIPrefix).Subrouter()

	// Health check and readiness endpoints
	api.HandleFunc("/health", s.healthHandler.Health).Methods("GET", "OPTIONS")
	api.HandleFunc("/info", s.healthHandler.Info).Methods("GET", "OPTIONS")
	api.HandleFunc("/ready", s.healthHandler.Ready).Methods("GET", "OPTIONS")
	api.HandleFunc("/live", s.healthHandler.Live).Methods("GET", "OPTIONS")

	// Bucket routes
	api.HandleFunc("/buckets", s.bucketHandler.List).Methods("GET", "OPTIONS")
	api.HandleFunc("/buckets/{bucket}", s.bucketHandler.Create).Methods("PUT", "OPTIONS")
	api.HandleFunc("/buckets/{bucket}", s.bucketHandler.Delete).Methods("DELETE", "OPTIONS")

	// Object routes
	api.HandleFunc("/objects/{bucket}", s.objectHandler.List).Methods("GET", "OPTIONS")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Upload).Methods("PUT", "OPTIONS")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Download).Methods("GET", "OPTIONS")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Head).Methods("HEAD", "OPTIONS")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Delete).Methods("DELETE", "OPTIONS")

	// Account management routes
	api.HandleFunc("/accounts", s.accountHandler.List).Methods("GET", "OPTIONS")
	api.HandleFunc("/accounts", s.accountHandler.Create).Methods("POST", "OPTIONS")
	api.HandleFunc("/accounts/{id}", s.accountHandler.Get).Methods("GET", "OPTIONS")
	api.HandleFunc("/accounts/{id}", s.accountHandler.Update).Methods("PUT", "OPTIONS")
	api.HandleFunc("/accounts/{id}", s.accountHandler.Delete).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/accounts/{id}/refresh", s.accountHandler.RefreshToken).Methods("POST", "OPTIONS")
	api.HandleFunc("/accounts/{id}/sync", s.accountHandler.SyncSpace).Methods("POST", "OPTIONS")

	// Space management routes
	api.HandleFunc("/space", s.spaceHandler.Overview).Methods("GET", "OPTIONS")
	api.HandleFunc("/space/accounts", s.spaceHandler.ListAccounts).Methods("GET", "OPTIONS")
	api.HandleFunc("/space/accounts/{id}", s.spaceHandler.AccountDetail).Methods("GET", "OPTIONS")
	api.HandleFunc("/space/accounts/{id}/sync", s.spaceHandler.SyncAccount).Methods("POST", "OPTIONS")

	// OAuth routes for OneDrive authorization
	api.HandleFunc("/oauth/setup", s.oauthHandler.SetupGuide).Methods("GET", "OPTIONS")
	api.HandleFunc("/oauth/create", s.oauthHandler.CreateAccount).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/oauth/accounts", s.oauthHandler.AccountList).Methods("GET", "OPTIONS")
	api.HandleFunc("/oauth/authorize/{id}", s.oauthHandler.Authorize).Methods("GET", "OPTIONS")
	api.HandleFunc("/oauth/callback", s.oauthHandler.Callback).Methods("GET", "OPTIONS")
	api.HandleFunc("/oauth/status/{id}", s.oauthHandler.TokenStatus).Methods("GET", "OPTIONS")

	// Virtual File System routes
	api.HandleFunc("/vfs/{bucket}/{path:.*}", s.vfsHandler.UploadFile).Methods("PUT", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/{path:.*}", s.vfsHandler.Get).Methods("GET", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/{path:.*}", s.vfsHandler.Head).Methods("HEAD", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/{path:.*}", s.vfsHandler.Delete).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_mkdir", s.vfsHandler.CreateDirectory).Methods("POST", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_move", s.vfsHandler.Move).Methods("POST", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_copy", s.vfsHandler.Copy).Methods("POST", "OPTIONS")

	// Enhanced VFS routes - Search
	api.HandleFunc("/vfs/{bucket}/_search", s.enhancedVFSHandler.Search).Methods("GET", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_files/recent", s.enhancedVFSHandler.GetRecentFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_files/by-date", s.enhancedVFSHandler.GetFilesByDateRange).Methods("GET", "OPTIONS")

	// Enhanced VFS routes - Starred files
	api.HandleFunc("/vfs/{bucket}/_starred", s.enhancedVFSHandler.GetStarredFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_starred", s.enhancedVFSHandler.StarFile).Methods("POST", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_starred/{file_id}", s.enhancedVFSHandler.UnstarFile).Methods("DELETE", "OPTIONS")

	// Enhanced VFS routes - Trash
	api.HandleFunc("/vfs/{bucket}/_trash", s.enhancedVFSHandler.GetTrashItems).Methods("GET", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_trash", s.enhancedVFSHandler.EmptyTrash).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_trash/{trash_id}/restore", s.enhancedVFSHandler.RestoreFromTrash).Methods("POST", "OPTIONS")
	api.HandleFunc("/vfs/{bucket}/_trash/{trash_id}", s.enhancedVFSHandler.DeleteFromTrash).Methods("DELETE", "OPTIONS")

	// Root endpoint - API info
	s.router.HandleFunc("/", s.healthHandler.Info).Methods("GET", "OPTIONS")
}
