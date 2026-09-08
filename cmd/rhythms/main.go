// Command rhythms runs the self-hosted habit-tracker server.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rhythms/internal/config"
	"rhythms/internal/reminder"
	"rhythms/internal/store"
	"rhythms/internal/web"
)

const shutdownTimeout = 5 * time.Second

// version and commit are set via -ldflags at release build time (see
// .goreleaser.yaml); "dev" when built directly with `go build`/`make build`.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()
	if *showVersion {
		slog.Info("rhythms", "version", version, "commit", commit)
		return
	}

	cfg := config.Load()
	if missing := cfg.Missing(); len(missing) > 0 {
		slog.Error("missing required config — auth is mandatory, there's no disabled mode", "missing", missing)
		os.Exit(1)
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	settings := store.NewSettingsRepo(db)
	vapidPublic, vapidPrivate, err := settings.EnsureVAPIDKeys(ctx)
	if err != nil {
		slog.Error("failed to set up VAPID keys", "error", err)
		os.Exit(1)
	}
	sessionSecret, err := settings.EnsureSessionSecret(ctx)
	if err != nil {
		slog.Error("failed to set up session secret", "error", err)
		os.Exit(1)
	}
	if err := store.NewUserRepo(db).EnsureAdmin(ctx, cfg.AdminEmail); err != nil {
		slog.Error("failed to ensure admin user", "error", err, "email", cfg.AdminEmail)
		os.Exit(1)
	}

	srv, err := web.NewServer(cfg, db, vapidPublic, sessionSecret)
	if err != nil {
		slog.Error("failed to build server", "error", err)
		os.Exit(1)
	}

	scheduler := reminder.NewScheduler(
		store.NewHabitRepo(db),
		store.NewEntryRepo(db),
		store.NewReminderRepo(db),
		store.NewReminderLogRepo(db),
		store.NewPushSubscriptionRepo(db),
		&reminder.WebPushSender{VAPIDPublicKey: vapidPublic, VAPIDPrivateKey: vapidPrivate, Subject: cfg.VAPIDSubject},
	)
	go scheduler.Run(ctx)

	httpServer := &http.Server{Addr: cfg.Addr, Handler: srv.Routes()}
	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	slog.Info("rhythms starting", "version", version, "addr", cfg.Addr, "db_path", cfg.DBPath, "base_url", cfg.BaseURL)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}
