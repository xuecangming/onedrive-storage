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
)

// Server represents the HTTP server
type Server struct {
	config         *types.Config
	db             *sql.DB
	router         *mux.Router
	bucketHandler  *handlers.BucketHandler
	objectHandler  *handlers.ObjectHandler
	accountHandler *handlers.AccountHandler
	spaceHandler   *handlers.SpaceHandler
	healthHandler  *handlers.HealthHandler
}

// NewServer creates a new HTTP server
func NewServer(config *types.Config, db *sql.DB) *Server {
	// Create repositories
	bucketRepo := repository.NewBucketRepository(db)
	objectRepo := repository.NewObjectRepository(db)
	accountRepo := repository.NewAccountRepository(db)

	// Create services
	bucketService := bucket.NewService(bucketRepo)
	accountService := account.NewService(accountRepo)
	objectService := object.NewService(objectRepo, bucketRepo)

	// Create handlers
	bucketHandler := handlers.NewBucketHandler(bucketService)
	objectHandler := handlers.NewObjectHandler(objectService)
	accountHandler := handlers.NewAccountHandler(accountService)
	spaceHandler := handlers.NewSpaceHandler(accountService)
	healthHandler := handlers.NewHealthHandler(db)

	server := &Server{
		config:         config,
		db:             db,
		router:         mux.NewRouter(),
		bucketHandler:  bucketHandler,
		objectHandler:  objectHandler,
		accountHandler: accountHandler,
		spaceHandler:   spaceHandler,
		healthHandler:  healthHandler,
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
	s.router.Use(middleware.LoggingMiddleware)
	s.router.Use(middleware.RecoveryMiddleware)

	// API v1 routes
	api := s.router.PathPrefix(s.config.Server.APIPrefix).Subrouter()

	// Health check
	api.HandleFunc("/health", s.healthHandler.Health).Methods("GET")
	api.HandleFunc("/info", s.healthHandler.Info).Methods("GET")

	// Bucket routes
	api.HandleFunc("/buckets", s.bucketHandler.List).Methods("GET")
	api.HandleFunc("/buckets/{bucket}", s.bucketHandler.Create).Methods("PUT")
	api.HandleFunc("/buckets/{bucket}", s.bucketHandler.Delete).Methods("DELETE")

	// Object routes
	api.HandleFunc("/objects/{bucket}", s.objectHandler.List).Methods("GET")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Upload).Methods("PUT")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Download).Methods("GET")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Head).Methods("HEAD")
	api.HandleFunc("/objects/{bucket}/{key}", s.objectHandler.Delete).Methods("DELETE")

	// Account management routes
	api.HandleFunc("/accounts", s.accountHandler.List).Methods("GET")
	api.HandleFunc("/accounts", s.accountHandler.Create).Methods("POST")
	api.HandleFunc("/accounts/{id}", s.accountHandler.Get).Methods("GET")
	api.HandleFunc("/accounts/{id}", s.accountHandler.Update).Methods("PUT")
	api.HandleFunc("/accounts/{id}", s.accountHandler.Delete).Methods("DELETE")
	api.HandleFunc("/accounts/{id}/refresh", s.accountHandler.RefreshToken).Methods("POST")
	api.HandleFunc("/accounts/{id}/sync", s.accountHandler.SyncSpace).Methods("POST")

	// Space management routes
	api.HandleFunc("/space", s.spaceHandler.Overview).Methods("GET")
	api.HandleFunc("/space/accounts", s.spaceHandler.ListAccounts).Methods("GET")
	api.HandleFunc("/space/accounts/{id}", s.spaceHandler.AccountDetail).Methods("GET")
	api.HandleFunc("/space/accounts/{id}/sync", s.spaceHandler.SyncAccount).Methods("POST")
}
