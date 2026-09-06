// Command rhythms runs the Rhythms habit-tracking web server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rhythms/internal/config"
	"rhythms/internal/push"
	"rhythms/internal/scheduler"
	"rhythms/internal/server"
	"rhythms/internal/store"
	"rhythms/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	renderer, err := server.NewRenderer(web.FS)
	if err != nil {
		return err
	}

	vapidPublic, vapidPrivate, err := push.LoadOrGenerateVAPIDKeys(db)
	if err != nil {
		return err
	}
	sender := push.NewSender(vapidPublic, vapidPrivate, cfg.VAPIDSubscriber)

	srv := server.New(db, renderer, web.FS, cfg, vapidPublic, sender)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sched := scheduler.New(db, sender)
	go sched.Run(ctx)

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
	}()

	slog.Info("listening", "addr", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
