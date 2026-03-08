package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/authvora/api-gateway/internal/config"
	"github.com/authvora/api-gateway/internal/middleware"
	"github.com/authvora/api-gateway/internal/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.LoadConfig()

	// Configure slog
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	// Connect to Redis (optional - rate limiter degrades gracefully)
	var redisClient *redis.Client
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Warn("redis URL parse failed, rate limiting disabled", "err", err)
	} else {
		redisClient = redis.NewClient(opt)
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			slog.Warn("redis connection failed, rate limiting disabled", "err", err)
			redisClient = nil
		}
	}

	r := chi.NewMux()

	// Global middleware
	r.Use(middleware.CORSMiddleware)
	r.Use(middleware.LoggingMiddleware)
	if redisClient != nil {
		r.Use(middleware.NewRateLimiter(redisClient))
	}

	// Proxies
	authProxy := proxy.NewServiceProxy(cfg.AuthServiceURL)
	riskProxy := proxy.NewServiceProxy(cfg.RiskEngineURL)
	policyProxy := proxy.NewServiceProxy(cfg.PolicyEngineURL)

	// Route definitions
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// /api/v1/auth/* - AuthServiceURL with TenantMiddleware only
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)
			r.Handle("/auth/*", authProxy)
		})

		// /api/v1/users/* - AuthServiceURL with TenantMiddleware + JWTAuthMiddleware
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)
			r.Use(middleware.JWTAuthMiddleware)
			r.Handle("/users/*", authProxy)
		})

		// /api/v1/risk/* - RiskEngineURL with TenantMiddleware + JWTAuthMiddleware
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)
			r.Use(middleware.JWTAuthMiddleware)
			r.Handle("/risk/*", riskProxy)
		})

		// /api/v1/policies/* - PolicyEngineURL with TenantMiddleware + JWTAuthMiddleware
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)
			r.Use(middleware.JWTAuthMiddleware)
			r.Handle("/policies/*", policyProxy)
		})

		// /api/v1/analytics/* - AuthServiceURL with TenantMiddleware + JWTAuthMiddleware
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)
			r.Use(middleware.JWTAuthMiddleware)
			r.Handle("/analytics/*", authProxy)
		})

		// /api/v1/simulate/* - RiskEngineURL with TenantMiddleware + JWTAuthMiddleware
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantMiddleware)
			r.Use(middleware.JWTAuthMiddleware)
			r.Handle("/simulate/*", riskProxy)
		})
	})

	addr := ":" + strconv.Itoa(cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		slog.Info("api gateway listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
