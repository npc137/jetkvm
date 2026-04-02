package main

import (
	"os"
)

// Config holds all runtime configuration for the management backend.
// Values are read from environment variables.
type Config struct {
	// ListenAddr is the address the HTTP server will bind to.
	// Default: ":8080"
	ListenAddr string

	// DBPath is the path to the SQLite database file.
	// Default: "/data/management.db"
	DBPath string

	// EntraTenantID is the Microsoft Entra (Azure AD) tenant ID.
	// Required.
	EntraTenantID string

	// EntraClientID is the application (client) ID registered in Entra.
	// Required.
	EntraClientID string

	// PublicBaseURL is the externally reachable base URL of this service,
	// used to construct connect URLs (e.g. "https://kvm.company.com").
	// Required.
	PublicBaseURL string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() Config {
	return Config{
		ListenAddr:    getEnv("LISTEN_ADDR", ":8080"),
		DBPath:        getEnv("DB_PATH", "/data/management.db"),
		EntraTenantID: mustGetEnv("ENTRA_TENANT_ID"),
		EntraClientID: mustGetEnv("ENTRA_CLIENT_ID"),
		PublicBaseURL: mustGetEnv("PUBLIC_BASE_URL"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable not set: " + key)
	}
	return v
}
