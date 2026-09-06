package auth

import (
	"context"
	"database/sql"
	"net/http"

	"rhythms/internal/store"
)

type contextKey int

const (
	userKey contextKey = iota
	sessionKey
)

func UserFromContext(ctx context.Context) *store.User {
	u, _ := ctx.Value(userKey).(*store.User)
	return u
}

func SessionFromContext(ctx context.Context) *store.Session {
	s, _ := ctx.Value(sessionKey).(*store.Session)
	return s
}

// RequireAuth loads the session/user for the request's session cookie,
// redirecting to /login (via a plain redirect, or an HX-Redirect header for
// htmx requests) if there is none.
func RequireAuth(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sid, ok := SessionIDFromRequest(r)
			if !ok {
				redirectToLogin(w, r)
				return
			}

			sess, err := store.GetSession(db, sid)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			user, err := store.GetUserByID(db, sess.UserID)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			_ = store.TouchSession(db, sid)

			ctx := context.WithValue(r.Context(), userKey, user)
			ctx = context.WithValue(ctx, sessionKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// CSRFProtect validates the X-CSRF-Token header against the session's stored
// token for every non-GET/HEAD/OPTIONS request. Must run after RequireAuth.
func CSRFProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		sess := SessionFromContext(r.Context())
		token := r.Header.Get(CSRFHeaderName)
		if token == "" {
			// Plain HTML form posts (no htmx, no JS) carry the token as a
			// hidden field instead of a header.
			token = r.FormValue("csrf_token")
		}
		if sess == nil || !ValidCSRFToken(sess.CSRFToken, token) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
