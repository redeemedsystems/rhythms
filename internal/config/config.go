// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config is the app's full runtime configuration, loaded once at startup
// from environment variables via Load — there's no config file.
type Config struct {
	Addr   string
	DBPath string

	// BaseURL is this deployment's own externally-reachable origin (e.g.
	// "https://rhythms.redeemed.systems", no trailing slash) — used to
	// build the exact OAuth redirect_uri sent to Google, which must match
	// what's registered in the Google Cloud Console, and to decide whether
	// the session cookie can be marked Secure.
	BaseURL string

	GoogleClientID     string
	GoogleClientSecret string

	// AdminEmail is ensured to exist as an admin user on every startup
	// (see store.UserRepo.EnsureAdmin) — the only bootstrap into an
	// otherwise invite-only system, and lockout recovery if that user's
	// row is ever deleted.
	AdminEmail string

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

// Load reads Config from environment variables, applying defaults for any
// that are unset.
func Load() Config {
	return Config{
		Addr:               envOr("RHYTHMS_ADDR", ":8080"),
		DBPath:             envOr("RHYTHMS_DB_PATH", "rhythms.db"),
		BaseURL:            strings.TrimSuffix(os.Getenv("RHYTHMS_BASE_URL"), "/"),
		GoogleClientID:     os.Getenv("RHYTHMS_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("RHYTHMS_GOOGLE_CLIENT_SECRET"),
		AdminEmail:         os.Getenv("RHYTHMS_ADMIN_EMAIL"),
		SkipEnabled:        envBoolOr("RHYTHMS_SKIP_ENABLED", true),
		VAPIDSubject:       envOr("RHYTHMS_VAPID_SUBJECT", "mailto:admin@localhost"),
	}
}

// CookieSecure reports whether the session cookie should be marked Secure
// (HTTPS-only) — derived from BaseURL's scheme rather than a separate flag,
// since the two must always agree: a Secure cookie is silently dropped by
// the browser on an http:// origin.
func (c Config) CookieSecure() bool {
	return strings.HasPrefix(c.BaseURL, "https://")
}

// Missing reports which required fields are unset, for a fail-fast startup
// check — auth is mandatory now (there's no "disabled" mode like Basic
// Auth had), so an incomplete config must never silently start an
// unreachable-by-design or insecure server.
func (c Config) Missing() []string {
	var missing []string
	if c.BaseURL == "" {
		missing = append(missing, "RHYTHMS_BASE_URL")
	}
	if c.GoogleClientID == "" {
		missing = append(missing, "RHYTHMS_GOOGLE_CLIENT_ID")
	}
	if c.GoogleClientSecret == "" {
		missing = append(missing, "RHYTHMS_GOOGLE_CLIENT_SECRET")
	}
	if c.AdminEmail == "" {
		missing = append(missing, "RHYTHMS_ADMIN_EMAIL")
	}
	return missing
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
