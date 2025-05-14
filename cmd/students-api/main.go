package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nansmatty/students-api-golang/internal/config"
	"github.com/nansmatty/students-api-golang/internal/http/handlers/students"
	"github.com/nansmatty/students-api-golang/internal/storage/sqlite"
)

func main() {
	// Load configuration --------------------------------------------------------------------------------
	cfg := config.MustLoad()

	// Connect to DB
	storage, err := sqlite.New(cfg)

	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}

	slog.Info("Connected to database", slog.String("env", cfg.Env))
	// Migrate DB

	// Setup router ---------------------------------------------------------------------------------------
	router := http.NewServeMux()

	router.HandleFunc("POST /api/students", students.New(storage))
	router.HandleFunc("GET /api/students/{id}", students.GetById(storage))

	// Setup server --------------------------------------------------------------------------------------
	server := http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	// Setup logging -------------------------------------------------------------------------------------
	// Slog is a structured logger for Go
	slog.Info("Server started on", slog.String("Address", cfg.Addr))

	// Setup graceful shutdown ---------------------------------------------------------------------------
	// Create a channel to listen for OS signals & this will allow us to gracefully shutdown the server --
	done := make(chan os.Signal, 1)

	// Notify the channel when an interrupt or terminate signal is received -----------------------------
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine ------------------------------------------------------------------
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal("Failed to start server")
		}
	}()

	// Wait for a signal to shutdown the server ----------------------------------------------------------
	<-done

	slog.Info("Shutting down server...")

	// Create a context with a timeout for the shutdown process ------------------------------------------
	// This will allow us to wait for a maximum of 5 seconds for the server to shutdown ------------------
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Ensure the cancel function is called to release resources -----------------------------------------
	// This is important to avoid memory leaks and to ensure that the context is properly cleaned up -----
	defer cancel()

	// Shutdown the server & this will close all connections and stop the server from accepting new requests
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown server", slog.String("error", err.Error()))
	}

	slog.Info("Server stopped successfully")
}
