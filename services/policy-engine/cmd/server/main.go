package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/authvora/policy-engine/internal/handler"
	"github.com/authvora/policy-engine/internal/repository"
	"github.com/authvora/policy-engine/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := loadConfig()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	repo := repository.NewPolicyRepository(pool)
	svc := service.NewPolicyService(repo)
	h := handler.NewPolicyHandler(svc)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1/policies", func(r chi.Router) {
		r.Get("/{tenantId}", h.HandleGetPolicy)
		r.Put("/{tenantId}", h.HandleUpdatePolicy)
		r.Post("/validate-password", h.HandleValidatePassword)
		r.Get("/{tenantId}/evaluate-mfa", h.HandleEvaluateMFA)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("policy-engine listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	log.Println("policy-engine stopped")
}

type config struct {
	DatabaseURL string
	Port        string
	LogLevel    string
}

func loadConfig() config {
	cfg := config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        "8083",
		LogLevel:    "info",
	}
	if p := os.Getenv("PORT"); p != "" {
		cfg.Port = p
	}
	if l := os.Getenv("LOG_LEVEL"); l != "" {
		cfg.LogLevel = l
	}
	return cfg
}
