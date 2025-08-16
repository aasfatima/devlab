package main

import (
	"context"
	"devlab/internal/cleanup"
	"devlab/internal/config"
	"devlab/internal/docker"
	"devlab/internal/storage"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("Async Cleanup Worker starting...")

	// Load configuration
	cfg := config.Load()
	log.Printf("[worker] configuration loaded: cleanup enabled=%v, interval=%v, max age=%v",
		cfg.Cleanup.EnableCleanup, cfg.Cleanup.CleanupInterval, cfg.Cleanup.MaxScenarioAge)

	// Connect to MongoDB
	mongoClient, err := storage.GetMongoClient(context.Background(), cfg.MongoURI)
	if err != nil {
		log.Fatalf("[worker] failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.DBName)
	log.Printf("[worker] connected to database: %s", cfg.DBName)

	// Initialize Docker client
	dockerClient := &docker.RealClient{}

	// Initialize cleanup manager
	cleanupManager := cleanup.NewCleanupManager(cfg, db, dockerClient)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start cleanup worker
	if cfg.Cleanup.EnableCleanup {
		log.Printf("[worker] starting cleanup worker with interval: %v", cfg.Cleanup.CleanupInterval)
		go func() {
			cleanupManager.RunPeriodicCleanup(ctx, cfg.Cleanup.CleanupInterval)
		}()
	} else {
		log.Println("[worker] cleanup is disabled")
	}

	// Wait for shutdown signal
	log.Println("[worker] cleanup worker running. Press Ctrl+C to stop.")
	<-sigChan
	log.Println("[worker] received shutdown signal, stopping cleanup worker...")

	// Give cleanup operations time to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Wait for shutdown context or timeout
	<-shutdownCtx.Done()
	log.Println("[worker] cleanup worker stopped")
}
