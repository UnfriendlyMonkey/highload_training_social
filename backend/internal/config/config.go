// Package config provides configuration for the application.
package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	return Config{
		HTTPAddr:    envOr("HTTP_ADDR", ":3000"),
		DatabaseURL: envOr("DATABASE_URL", "postgres://someuser:somepass@localhost:5435/socialnet?sslmode=disable"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
