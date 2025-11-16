package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wallet-ledger/handlers"
	"wallet-ledger/repository"
	"wallet-ledger/service"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get configuration from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to database")

	// Initialize layers
	repo := repository.New(db)
	svc := service.New(repo)
	handler := handlers.New(svc, repo)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Running idempotency key cleanup...")
				if err := repo.CleanupOldIdempotencyKeys(); err != nil {
					log.Printf("Error cleaning up idempotency keys: %v", err)
				} else {
					log.Println("Idempotency key cleanup completed")
				}
			case <-ctx.Done():
				log.Println("Stopping cleanup goroutine...")
				return
			}
		}
	}()

	// Setup routes
	router := handler.SetupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Cancel context to stop background goroutines
	cancel()

	// Give the server 30 seconds to finish current requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
