package config

import (
	"os"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	Environment        string
	LogLevel           string
	EntraTenantID      string
	EntraClientID      string
	CORSAllowedOrigins []string
}

func Load() Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		host := getEnv("DB_HOST", "localhost")
		user := getEnv("DB_USER", "postgres")
		pass := getEnv("DB_PASSWORD", "postgres")
		name := getEnv("DB_NAME", "healthcheck")
		// Default to require for remote databases unless DB_SSLMODE is set
		ssl := getEnv("DB_SSLMODE", "require")
		if host == "localhost" {
			ssl = "disable"
		}
		dbURL = "postgres://" + user + ":" + pass + "@" + host + ":5432/" + name + "?sslmode=" + ssl
	}

	originsRaw := os.Getenv("CORS_ALLOWED_ORIGINS")
	var origins []string
	if originsRaw != "" {
		for _, o := range strings.Split(originsRaw, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}

	return Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        dbURL,
		Environment:        getEnv("ENV", "production"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		EntraTenantID:      os.Getenv("ENTRA_TENANT_ID"),
		EntraClientID:      os.Getenv("ENTRA_CLIENT_ID"),
		CORSAllowedOrigins: origins,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
