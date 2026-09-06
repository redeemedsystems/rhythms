// Package config parses runtime configuration from flags and environment variables.
package config

import (
	"flag"
	"os"
)

type Config struct {
	Addr            string
	DBPath          string
	SecureCookies   bool
	Debug           bool
	VAPIDSubscriber string
}

func Load() Config {
	cfg := Config{
		Addr:            envOr("RHYTHMS_ADDR", ":8080"),
		DBPath:          envOr("RHYTHMS_DB", "data/rhythms.db"),
		SecureCookies:   envOr("RHYTHMS_SECURE_COOKIES", "true") == "true",
		Debug:           envOr("RHYTHMS_DEBUG", "false") == "true",
		VAPIDSubscriber: envOr("RHYTHMS_VAPID_SUBSCRIBER", "mailto:admin@localhost"),
	}

	flag.StringVar(&cfg.Addr, "addr", cfg.Addr, "listen address")
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "path to the sqlite database file")
	flag.BoolVar(&cfg.SecureCookies, "secure-cookies", cfg.SecureCookies, "set the Secure flag on session cookies (disable only for plain-HTTP LAN use)")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "enable debug-only routes (e.g. test push notifications)")
	flag.StringVar(&cfg.VAPIDSubscriber, "vapid-subscriber", cfg.VAPIDSubscriber, "contact URI (mailto: or https:) sent in the VAPID JWT")
	flag.Parse()

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
