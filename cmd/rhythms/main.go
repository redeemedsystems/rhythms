// Command rhythms runs the self-hosted habit-tracker server.
package main

import (
	"context"
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

func main() {
	cfg := config.Load()

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

	srv, err := web.NewServer(cfg, db, vapidPublic)
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

	slog.Info("rhythms starting", "addr", cfg.Addr, "db_path", cfg.DBPath, "auth_enabled", cfg.AuthEnabled())
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}
