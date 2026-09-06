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
	BaseURL         string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

func Load() Config {
	cfg := Config{
		Addr:            envOr("RHYTHMS_ADDR", ":8080"),
		DBPath:          envOr("RHYTHMS_DB", "data/rhythms.db"),
		SecureCookies:   envOr("RHYTHMS_SECURE_COOKIES", "true") == "true",
		Debug:           envOr("RHYTHMS_DEBUG", "false") == "true",
		VAPIDSubscriber: envOr("RHYTHMS_VAPID_SUBSCRIBER", "mailto:admin@localhost"),
		BaseURL:         envOr("RHYTHMS_BASE_URL", "http://localhost:8080"),

		SMTPHost:     envOr("RHYTHMS_SMTP_HOST", ""),
		SMTPPort:     envOr("RHYTHMS_SMTP_PORT", "587"),
		SMTPUsername: envOr("RHYTHMS_SMTP_USERNAME", ""),
		// Password is env-only (no flag): flag values are visible in the
		// process list, which a credential shouldn't be.
		SMTPPassword: envOr("RHYTHMS_SMTP_PASSWORD", ""),
		SMTPFrom:     envOr("RHYTHMS_SMTP_FROM", "Rhythms <no-reply@localhost>"),
	}

	flag.StringVar(&cfg.Addr, "addr", cfg.Addr, "listen address")
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "path to the sqlite database file")
	flag.BoolVar(&cfg.SecureCookies, "secure-cookies", cfg.SecureCookies, "set the Secure flag on session cookies (disable only for plain-HTTP LAN use)")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "enable debug-only routes (e.g. test push notifications)")
	flag.StringVar(&cfg.VAPIDSubscriber, "vapid-subscriber", cfg.VAPIDSubscriber, "contact URI (mailto: or https:) sent in the VAPID JWT")
	flag.StringVar(&cfg.BaseURL, "base-url", cfg.BaseURL, "public base URL used to build links in emails (e.g. https://rhythms.example.com)")
	flag.StringVar(&cfg.SMTPHost, "smtp-host", cfg.SMTPHost, "SMTP server host for sending email (leave empty to log emails instead of sending)")
	flag.StringVar(&cfg.SMTPPort, "smtp-port", cfg.SMTPPort, "SMTP server port")
	flag.StringVar(&cfg.SMTPUsername, "smtp-username", cfg.SMTPUsername, "SMTP auth username")
	flag.StringVar(&cfg.SMTPFrom, "smtp-from", cfg.SMTPFrom, "From address used for outgoing email")
	flag.Parse()

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
