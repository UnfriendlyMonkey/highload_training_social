// Package config provides configuration for the application.
package config

import "os"

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	DatabaseURLMaster   string
	DatabaseURLReplica1 string
	DatabaseURLReplica2 string
}

func Load() Config {
	return Config{
		HTTPAddr:            envOr("HTTP_ADDR", ":3000"),
		DatabaseURL:         envOr("DATABASE_URL", "postgres://someuser:somepass@localhost:5435/socialnet?sslmode=disable"),
		DatabaseURLMaster:   envOr("DATABASE_URL_MASTER", "postgres://someuser:somepass@localhost:5436/socialnet?sslmode=disable"),
		// DatabaseURLReplica1: envOr("DATABASE_URL_REPLICA_1", "postgres://someuser:somepass@localhost:5436/socialnet?sslmode=disable"),
		DatabaseURLReplica2: envOr("DATABASE_URL_REPLICA_2", "postgres://someuser:somepass@localhost:5437/socialnet?sslmode=disable"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
