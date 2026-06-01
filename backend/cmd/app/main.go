package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UnfriendlyMonkey/hsn/internal/api/http/resource"
	"github.com/UnfriendlyMonkey/hsn/internal/config"
	"github.com/UnfriendlyMonkey/hsn/internal/service"
	"github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepo(db)

	userSvc := service.NewUserService(userRepo)

	r := resource.NewRouter(userSvc)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	log.Println("server stopped")
}
