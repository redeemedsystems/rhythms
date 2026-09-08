package web

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"
)

// basicAuth gates every request behind a single shared username/password,
// matching the "single-user, home-network" auth model — no sessions, no
// per-user accounts. It's a no-op wrapper when auth isn't configured.
func basicAuth(user, pass string, next http.Handler) http.Handler {
	if user == "" && pass == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(gotUser), []byte(user)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(gotPass), []byte(pass)) == 1
		if !ok || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="rhythms"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
