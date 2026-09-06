// Command rhythms runs the self-hosted habit-tracker server.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"rhythms/internal/config"
	"rhythms/internal/store"
	"rhythms/internal/web"
)

func main() {
	cfg := config.Load()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	srv, err := web.NewServer(cfg, db)
	if err != nil {
		slog.Error("failed to build server", "error", err)
		os.Exit(1)
	}

	slog.Info("rhythms starting", "addr", cfg.Addr, "db_path", cfg.DBPath, "auth_enabled", cfg.AuthEnabled())
	if err := http.ListenAndServe(cfg.Addr, srv.Routes()); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}
