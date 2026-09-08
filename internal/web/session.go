package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	sessionCookieName = "rhythms_session"
	sessionDuration   = 30 * 24 * time.Hour
)

// signSession produces a cookie value of the form
// "<userID>.<unixExpiry>.<signature>" — an HMAC-SHA256 over the
// "<userID>.<unixExpiry>" payload, so a tampered userID or a pushed-out
// expiry both fail verification. No session store: the payload is the
// entire state, and requireAuth re-fetches the User row fresh from the DB
// on every request, so revocation (an admin deleting a user) takes effect
// immediately rather than waiting for the cookie to expire.
func signSession(secret []byte, userID int64, exp time.Time) string {
	payload := fmt.Sprintf("%d.%d", userID, exp.Unix())
	return payload + "." + sign(secret, payload)
}

// verifySession parses and validates a cookie value, returning the userID
// it names if the signature is valid and it hasn't expired.
func verifySession(secret []byte, value string) (userID int64, ok bool) {
	idx := strings.LastIndex(value, ".")
	if idx < 0 {
		return 0, false
	}
	payload, sig := value[:idx], value[idx+1:]
	if !hmac.Equal([]byte(sig), []byte(sign(secret, payload))) {
		return 0, false
	}

	parts := strings.SplitN(payload, ".", 2)
	if len(parts) != 2 {
		return 0, false
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	expUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	if time.Now().Unix() > expUnix {
		return 0, false
	}
	return userID, true
}

func sign(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) setSessionCookie(w http.ResponseWriter, userID int64) {
	exp := time.Now().Add(sessionDuration)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signSession(s.sessionSecret, userID, exp),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
