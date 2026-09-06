// Package config loads runtime configuration from environment variables.
package config

import "os"

type Config struct {
	Addr          string
	DBPath        string
	BasicAuthUser string
	BasicAuthPass string
}

func Load() Config {
	return Config{
		Addr:          envOr("RHYTHMS_ADDR", ":8080"),
		DBPath:        envOr("RHYTHMS_DB_PATH", "rhythms.db"),
		BasicAuthUser: os.Getenv("RHYTHMS_BASIC_AUTH_USER"),
		BasicAuthPass: os.Getenv("RHYTHMS_BASIC_AUTH_PASS"),
	}
}

// AuthEnabled reports whether basic-auth gating should be applied. Auth is
// off by default so a single-user home-network deployment needs no setup.
func (c Config) AuthEnabled() bool {
	return c.BasicAuthUser != "" && c.BasicAuthPass != ""
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
