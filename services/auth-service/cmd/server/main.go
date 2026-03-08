package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/authvora/auth-service/internal/handler"
	"github.com/authvora/auth-service/internal/jwt"
	"github.com/authvora/auth-service/internal/middleware"
	"github.com/authvora/auth-service/internal/oauth"
	"github.com/authvora/auth-service/internal/repository"
	"github.com/authvora/auth-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := loadConfig()

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("warning: failed to parse REDIS_URL, falling back to localhost:6379: %v", err)
		redisOpts = &redis.Options{Addr: "localhost:6379"}
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("warning: redis ping failed: %v", err)
	}
	defer rdb.Close()

	userRepo := repository.NewUserRepository(pool)
	credRepo := repository.NewCredentialRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	refreshRepo := repository.NewRefreshTokenRepository(pool)
	signingRepo := repository.NewSigningKeyRepository(pool)
	eventRepo := repository.NewEventRepository(pool)
	oauthRepo := repository.NewOAuthConnectionRepository(pool)
	tenantRepo := repository.NewTenantRepository(pool)

	jwtManager := jwt.NewManager(cfg.JWTIssuer, signingRepo)
	googleOAuth := oauth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret)

	authService := service.NewAuthService(
		userRepo, credRepo, sessionRepo, refreshRepo,
		eventRepo, oauthRepo, tenantRepo,
		jwtManager, googleOAuth,
	)

	authHandler := handler.NewAuthHandler(authService, jwtManager)
	userHandler := handler.NewUserHandler(authService)
	analyticsHandler := handler.NewAnalyticsHandler(eventRepo)

	r := chi.NewRouter()
	r.Use(middleware.TenantMiddleware)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.HandleRegister)
		r.Post("/login", authHandler.HandleLogin)
		r.Post("/oauth/google", authHandler.HandleOAuthGoogle)
		r.Post("/refresh", authHandler.HandleRefreshToken)
		r.Post("/revoke", authHandler.HandleRevokeToken)
		r.Get("/jwks", authHandler.HandleJWKS)
	})

	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Get("/me", userHandler.HandleGetCurrentUser)
		r.Get("/{id}", userHandler.HandleGetUserByID)
	})

	r.Route("/analytics", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Get("/login-metrics", analyticsHandler.HandleGetLoginMetrics)
		r.Get("/suspicious-ips", analyticsHandler.HandleGetSuspiciousIPs)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("server stopped")
}

type config struct {
	DatabaseURL      string
	RedisURL         string
	Port             string
	JWTIssuer        string
	GoogleClientID   string
	GoogleClientSecret string
	LogLevel         string
}

func loadConfig() config {
	return config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://authvora:authvora_secret@localhost:5432/authvora?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://:authvora_redis@localhost:6379/0"),
		Port:              getEnv("PORT", "8081"),
		JWTIssuer:         getEnv("JWT_ISSUER", "authvora"),
		GoogleClientID:    getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
