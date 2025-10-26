package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dejobratic/tbd/internal/config"
	"github.com/dejobratic/tbd/internal/database"
	idempostgres "github.com/dejobratic/tbd/internal/idempotency/postgres"
	"github.com/dejobratic/tbd/internal/kafka"
	httpadapter "github.com/dejobratic/tbd/internal/orders/adapters/http"
	orderspostgres "github.com/dejobratic/tbd/internal/orders/adapters/postgres"
	ordersapp "github.com/dejobratic/tbd/internal/orders/app"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		logger.Info("running database migrations", "path", cfg.MigrationsPath)
		if err := database.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
			logger.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations completed successfully")
	}

	repo := orderspostgres.NewRepository(pool)
	idemStore := idempostgres.NewStore(pool)
	eventBus := kafka.NewNoopEventBus()

	service := ordersapp.NewService(repo, eventBus, idemStore)
	ordersHandler := httpadapter.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := database.CheckHealth(r.Context(), pool); err != nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc(cfg.MetricsPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# metrics are not yet implemented\n"))
	})

	ordersHandler.Register(mux)

	handler := withRecovery(withLogging(mux))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("http server starting", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownGrace)*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	} else {
		logger.Info("http server stopped")
	}
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("http request", "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", time.Since(start))
	})
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec)
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
