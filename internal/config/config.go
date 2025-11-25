package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	Port        int
}

func FromEnv() Config {
	port := 8080
	if value := os.Getenv("PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			port = parsed
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://app:app@localhost:5432/app?sslmode=disable"
	}

	return Config{
		DatabaseURL: dbURL,
		Port:        port,
	}
}
