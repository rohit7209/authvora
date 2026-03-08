package config

import (
	"os"
	"strconv"
)

// Config holds the API Gateway configuration loaded from environment.
type Config struct {
	Port            int
	AuthServiceURL  string
	RiskEngineURL   string
	PolicyEngineURL string
	RedisURL        string
	LogLevel        string
}

// LoadConfig reads configuration from environment variables with defaults.
func LoadConfig() *Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	return &Config{
		Port:            port,
		AuthServiceURL:  getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		RiskEngineURL:   getEnv("RISK_ENGINE_URL", "http://localhost:8082"),
		PolicyEngineURL: getEnv("POLICY_ENGINE_URL", "http://localhost:8083"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379/1"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
