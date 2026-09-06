// Package mail sends transactional email (currently just password reset
// links) over SMTP.
package mail

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"rhythms/internal/config"
)

// Sender delivers a plain-text email.
type Sender interface {
	Send(to, subject, body string) error
}

// NewFromConfig builds a Sender from cfg. If no SMTP host is configured, it
// falls back to logging the message instead of failing outright, so the app
// still runs (and reset links are still recoverable, from the logs) before
// SMTP is set up.
func NewFromConfig(cfg config.Config) Sender {
	if cfg.SMTPHost == "" {
		return LogSender{}
	}
	return &SMTPSender{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	}
}

// SMTPSender sends mail through an SMTP relay, upgrading to TLS via STARTTLS
// when the server offers it and authenticating with PlainAuth when a
// username is set.
type SMTPSender struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (s *SMTPSender) Send(to, subject, body string) error {
	addr := s.Host + ":" + s.Port

	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s\r\n",
		s.From, to, subject, body,
	)

	return smtp.SendMail(addr, auth, s.From, []string{to}, []byte(msg))
}

// LogSender logs the email instead of sending it, for environments without
// SMTP configured.
type LogSender struct{}

func (LogSender) Send(to, subject, body string) error {
	slog.Warn("SMTP not configured; logging email instead of sending", "to", to, "subject", subject, "body", body)
	return nil
}
