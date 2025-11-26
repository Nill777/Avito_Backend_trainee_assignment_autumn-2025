package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"reviewer/internal/config"
	"reviewer/internal/handler"
	"reviewer/internal/logger"
	"reviewer/internal/repository/postgres"
	"reviewer/internal/service"
	"reviewer/migrations"
)

func main() {
	cfg := config.FromEnv()
	log := logger.New()

	log.Info("starting application", "port", cfg.Port)

	if err := migrations.Run(cfg.DatabaseURL); err != nil {
		log.Error("failed to run migrations", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		return
	}

	svc := service.New(repo)
	h := handler.New(svc, log)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "openapi.yaml")
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/openapi.yaml"),
	))

	h.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("server listening", "addr", addr)
		log.Info("documentation available at", "url", fmt.Sprintf("http://localhost:%d/swagger/index.html", cfg.Port))

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	if closer, ok := repo.(interface{ Close() }); ok {
		closer.Close()
	}
	log.Info("server exited properly")
}
