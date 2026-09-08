package web

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"rhythms/internal/domain"
)

type contextKey int

const userContextKey contextKey = iota

// isPublicPath reports whether a request can proceed with no session at
// all — the login flow itself (or it could never be reached), plus the
// handful of routes that were already effectively public before this
// (health checks, the PWA manifest/service worker, static assets, whose
// CSS the login page itself needs to load).
func isPublicPath(path string) bool {
	switch path {
	case "/login", "/auth/google/callback", "/logout",
		"/healthz", "/manifest.webmanifest", "/sw.js":
		return true
	}
	return strings.HasPrefix(path, "/static/")
}

// requireAuth resolves the session cookie into a *domain.User in the
// request context on every request (even public ones — so handleLoginPage
// can bounce an already-signed-in visitor straight to "/"), but only
// enforces having one on non-public paths. The user is re-fetched from the
// database every time rather than trusted from the cookie payload, so an
// admin deleting a user takes effect on that user's very next request
// rather than waiting for their cookie to expire.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			if userID, ok := verifySession(s.sessionSecret, cookie.Value); ok {
				if u, err := s.users.Get(r.Context(), userID); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), userContextKey, &u))
				}
			}
		}

		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if userFromContext(r.Context()) == nil {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireAdmin gates a route on the current user being an admin. It only
// needs to check the flag, not authenticate — requireAuth (wrapping the
// whole mux) already guarantees a non-nil user by the time any registered
// handler runs.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := userFromContext(r.Context()); u == nil || !u.IsAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func userFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userContextKey).(*domain.User)
	return u
}

// mustUser returns the current user, panicking if there isn't one — safe
// to call from any handler reachable only through requireAuth (i.e. every
// handler registered on a non-public path), which is all of them; the
// panic is a loud signal that a new route was wired up wrong, caught by
// recoverer rather than silently operating as no one.
func mustUser(r *http.Request) domain.User {
	u := userFromContext(r.Context())
	if u == nil {
		panic("mustUser called on a request with no session — this route must be reachable only through requireAuth")
	}
	return *u
}

// cacheStatic sets a bounded cache lifetime on embedded static assets.
// http.FileServerFS can't offer a real validator here — go:embed files carry
// a zero ModTime, so http.ServeContent never emits Last-Modified/ETag or
// handles conditional requests for them — and asset URLs aren't
// content-hashed, so a max-age longer than the gap between deploys risks a
// browser holding stale CSS/JS past a real change. An hour bounds that
// staleness to something a redeploy is very unlikely to land inside, while
// still skipping a full re-fetch on every page view within a session.
func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "path", r.URL.Path)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
