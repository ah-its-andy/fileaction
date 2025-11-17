package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andi/fileaction/backend/api"
	"github.com/andi/fileaction/backend/config"
	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/executor"
	"github.com/andi/fileaction/backend/scheduler"
	"github.com/andi/fileaction/backend/watcher"
)

func main() {
	// Load configuration
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config/config.yaml"
	}

	cfg, err := config.LoadFromEnv(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	if err := os.MkdirAll(cfg.Logging.Dir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logFile, err := os.OpenFile(cfg.Logging.AppLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// 设置日志同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	log.Println("=== FileAction Starting ===")
	log.Printf("Configuration: %+v", cfg)

	// Initialize database
	// cfg.Database.Path supports both SQLite and MySQL:
	// - SQLite: "./data/fileaction.db" or any path ending with .db
	// - MySQL: "user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Reset running tasks to pending status on startup
	taskRepo := database.NewTaskRepo(db)
	resetCount, err := taskRepo.ResetRunningTasks()
	if err != nil {
		log.Printf("Warning: Failed to reset running tasks: %v", err)
	} else if resetCount > 0 {
		log.Printf("Reset %d running task(s) to pending status", resetCount)
	}

	// Initialize executor
	exec := executor.New(
		db,
		cfg.Logging.Dir,
		cfg.Execution.TaskTimeout,
		cfg.Execution.StepTimeout,
	)
	log.Println("Executor initialized")

	// Initialize scheduler
	sched := scheduler.New(
		db,
		exec,
		cfg.Scheduler.MaxRunning,
		cfg.Scheduler.ScanInterval,
	)
	sched.Start()
	defer sched.Stop()
	log.Println("Scheduler initialized and started")

	// Initialize file watcher
	watch, err := watcher.New(db)
	if err != nil {
		log.Fatalf("Failed to initialize file watcher: %v", err)
	}
	// Start file watcher asynchronously
	go func() {
		if err := watch.Start(); err != nil {
			log.Printf("File watcher error: %v", err)
		}
	}()
	defer watch.Stop()
	log.Println("File watcher initialized and started")

	// Initialize API server
	server := api.New(db, sched, watch, cfg.Logging.Dir)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s", addr)
		fmt.Printf("FileAction server is running on http://%s\n", addr)
		if err := server.Start(addr); err != nil {
			serverErrors <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case sig := <-quit:
		log.Printf("Received signal: %v", sig)
		log.Println("Shutting down gracefully...")

		// Create a deadline for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown server
		log.Println("Stopping HTTP server...")
		if err := server.Shutdown(); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}

		// Stop file watcher
		log.Println("Stopping file watcher...")
		watch.Stop()

		// Stop scheduler (this will wait for running tasks to complete or timeout)
		log.Println("Stopping scheduler...")
		sched.Stop()

		// Close database connections
		log.Println("Closing database connections...")
		db.Close()

		// Wait for context deadline or completion
		<-ctx.Done()
		log.Println("Shutdown complete")
	}
}
