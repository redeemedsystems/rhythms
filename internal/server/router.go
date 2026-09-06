package server

import (
	"database/sql"
	"io/fs"
	"net/http"

	"rhythms/internal/auth"
	"rhythms/internal/config"
	"rhythms/internal/mail"
	"rhythms/internal/push"
)

type Server struct {
	db             *sql.DB
	render         *Renderer
	cfg            config.Config
	webFS          fs.FS
	vapidPublicKey string
	sender         *push.Sender
	mailer         mail.Sender
}

func New(db *sql.DB, render *Renderer, webFS fs.FS, cfg config.Config, vapidPublicKey string, sender *push.Sender, mailer mail.Sender) *Server {
	return &Server{
		db:             db,
		render:         render,
		cfg:            cfg,
		webFS:          webFS,
		vapidPublicKey: vapidPublicKey,
		sender:         sender,
		mailer:         mailer,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.SessionIDFromRequest(r); ok {
			http.Redirect(w, r, "/today", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	mux.HandleFunc("GET /register", s.handleRegisterForm)
	mux.HandleFunc("POST /register", s.handleRegister)
	mux.HandleFunc("GET /login", s.handleLoginForm)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /forgot-password", s.handleForgotPasswordForm)
	mux.HandleFunc("POST /forgot-password", s.handleForgotPassword)
	mux.HandleFunc("GET /reset-password", s.handleResetPasswordForm)
	mux.HandleFunc("POST /reset-password", s.handleResetPassword)

	mux.HandleFunc("GET /manifest.json", serveEmbedded(s.webFS, "static/manifest.json", "application/manifest+json"))
	mux.HandleFunc("GET /sw.js", serveEmbedded(s.webFS, "static/sw.js", "application/javascript"))
	mux.HandleFunc("GET /api/vapid-public-key", s.handleVAPIDPublicKey)

	staticSub, err := fs.Sub(s.webFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))

	htmxSub, err := fs.Sub(s.webFS, "htmx")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /htmx/", http.StripPrefix("/htmx/", http.FileServerFS(htmxSub)))

	authed := http.NewServeMux()
	authed.HandleFunc("POST /logout", s.handleLogout)

	authed.HandleFunc("GET /today", s.handleToday)
	authed.HandleFunc("POST /habits/{id}/log", s.handleHabitLog)

	authed.HandleFunc("GET /habits", s.handleHabitsList)
	authed.HandleFunc("GET /habits/new", s.handleHabitNewForm)
	authed.HandleFunc("POST /habits/new", s.handleHabitCreate)
	authed.HandleFunc("GET /habits/{id}/edit", s.handleHabitEditForm)
	authed.HandleFunc("POST /habits/{id}/edit", s.handleHabitUpdate)
	authed.HandleFunc("POST /habits/{id}/delete", s.handleHabitArchive)

	authed.HandleFunc("GET /dashboard", s.handleDashboard)
	authed.HandleFunc("GET /dashboard/partial", s.handleDashboardPartial)

	authed.HandleFunc("GET /habits/{id}/day", s.handleHabitDay)
	authed.HandleFunc("POST /habits/{id}/day", s.handleHabitDayLog)
	authed.HandleFunc("POST /habits/{id}/day/logs/{logID}/delete", s.handleHabitDayLogDelete)

	authed.HandleFunc("POST /push/subscribe", s.handlePushSubscribe)
	authed.HandleFunc("POST /push/unsubscribe", s.handlePushUnsubscribe)

	if s.cfg.Debug {
		authed.HandleFunc("POST /debug/test-push", s.handleDebugTestPush)
	}

	mux.Handle("/", chain(authed, auth.RequireAuth(s.db), auth.CSRFProtect))

	return mux
}

func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
