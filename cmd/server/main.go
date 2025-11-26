package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/api"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/database"
)

func main() {
	// Load configuration
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresDB(config.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create API server
	server := api.NewServer(config, db)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.Router(),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
