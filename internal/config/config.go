// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Addr          string
	DBPath        string
	BasicAuthUser string
	BasicAuthPass string

	// SkipEnabled gates whether the checkmark click-cycle can reach
	// SKIP/UNKNOWN states, or only cycles between No/YesManual. A
	// deployment-wide preference rather than per-habit, matching how
	// uHabits itself exposes this as a single app-level setting.
	SkipEnabled bool

	// VAPIDSubject identifies the sender to push services (a mailto: or
	// https: contact), as required by the Web Push spec. No default — an
	// operator-specific contact is the point, so it must be set explicitly
	// for reminders to actually send.
	VAPIDSubject string
}

func Load() Config {
	return Config{
		Addr:          envOr("RHYTHMS_ADDR", ":8080"),
		DBPath:        envOr("RHYTHMS_DB_PATH", "rhythms.db"),
		BasicAuthUser: os.Getenv("RHYTHMS_BASIC_AUTH_USER"),
		BasicAuthPass: os.Getenv("RHYTHMS_BASIC_AUTH_PASS"),
		SkipEnabled:   envBoolOr("RHYTHMS_SKIP_ENABLED", true),
		VAPIDSubject:  envOr("RHYTHMS_VAPID_SUBJECT", "mailto:admin@localhost"),
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

func envBoolOr(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
