// Package config reads runtime settings from the environment.
package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr    string // listen address, e.g. localhost:8080
	DataDir string // where state.json lives (Plaid mode only)
	UIDir   string // built UI to serve

	PlaidClientID string
	PlaidSecret   string
	PlaidEnv      string // sandbox | production
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		Addr:          getenv("FINANCESITE_ADDR", "localhost:8080"),
		DataDir:       getenv("FINANCESITE_DATA_DIR", "./data/local"),
		UIDir:         getenv("FINANCESITE_UI_DIR", "./ui/dist"),
		PlaidClientID: getenv("PLAID_CLIENT_ID", ""),
		PlaidSecret:   getenv("PLAID_SECRET", ""),
		PlaidEnv:      strings.ToLower(getenv("PLAID_ENV", "sandbox")),
	}
}

// PlaidConfigured reports whether real bank connections are possible.
// Without credentials the server runs in demo mode on sample data.
func (c Config) PlaidConfigured() bool {
	return c.PlaidClientID != "" && c.PlaidSecret != ""
}
