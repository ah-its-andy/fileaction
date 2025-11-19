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
	// cfg.Database.Path now should be MySQL DSN format: user:password@tcp(host:port)/dbname?params
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Reset any running tasks to pending (handles interrupted tasks from previous run)
	taskRepo := database.NewTaskRepo(db)
	if resetCount, err := taskRepo.ResetRunningTasks(); err != nil {
		log.Printf("Warning: Failed to reset running tasks: %v", err)
	} else if resetCount > 0 {
		log.Printf("Reset %d running task(s) to pending status", resetCount)
	}

	// Initialize task scheduler with integrated executor pool
	sched := scheduler.New(
		db,
		cfg.Execution.DefaultConcurrency,
		2*time.Second,
		cfg.Logging.Dir,
		cfg.Execution.TaskTimeout,
		cfg.Execution.StepTimeout,
	)
	sched.Start()
	defer sched.Stop()
	log.Printf("Task scheduler initialized with %d executors", cfg.Execution.DefaultConcurrency)

	// Initialize file watcher
	watch, err := watcher.New(db)
	if err != nil {
		log.Fatalf("Failed to initialize file watcher: %v", err)
	}
	if err := watch.Start(); err != nil {
		log.Fatalf("Failed to start file watcher: %v", err)
	}
	defer watch.Stop()
	log.Println("File watcher initialized and started")

	// Initialize API server
	server := api.New(db, sched, watch, cfg.Logging.Dir)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Connect scheduler to WebSocket hub for real-time log broadcasting
	sched.SetWebSocketHub(server.GetWebSocketHub())

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

		// Stop scheduler (this will wait for running tasks to complete)
		log.Println("Stopping scheduler...")
		sched.Stop()

		// Stop watcher
		log.Println("Stopping watcher...")
		watch.Stop()

		// Close database connections
		log.Println("Closing database connections...")
		db.Close()

		// Wait for context deadline or completion
		<-ctx.Done()
		log.Println("Shutdown complete")
	}
}
