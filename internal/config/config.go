package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	Environment string
	LogLevel    string
}

func Load() Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		host := getEnv("DB_HOST", "localhost")
		user := getEnv("DB_USER", "postgres")
		pass := getEnv("DB_PASSWORD", "postgres")
		name := getEnv("DB_NAME", "healthcheck")
		// Use sslmode=require for Azure
		ssl := "disable"
		if host != "localhost" {
			ssl = "require"
		}
		dbURL = "postgres://" + user + ":" + pass + "@" + host + ":5432/" + name + "?sslmode=" + ssl
	}

	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: dbURL,
		Environment: getEnv("ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
