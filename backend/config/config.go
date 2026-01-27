package config

import (
	"os"
)

type Config struct {
	ServerPort    string
	DatabaseURL   string
	JWTSecret     string
	StoragePath   string
	MaxUploadSize int64
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/stratus?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		StoragePath:   getEnv("STORAGE_PATH", "./storage"),
		MaxUploadSize: 1024 * 1024 * 1024 * 1024 * 1024,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
