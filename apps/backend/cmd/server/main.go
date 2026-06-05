// Command server starts the fabula-studio backend HTTP service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabula-studio/backend/internal/config"
	"github.com/fabula-studio/backend/internal/server"
)

func main() {
	cfg := config.Load()

	srv := server.New(cfg)

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	sig := <-quit
	log.Printf("received signal %v, shutting down...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped gracefully")
}
