package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"rhythms/internal/config"
)

// googleIdentity is what we need out of a completed Google sign-in — never
// more (no name, no picture, nothing that isn't used to authorize or
// identify the account).
type googleIdentity struct {
	Email         string
	EmailVerified bool
	Sub           string // Google's stable, unique account id
}

// googleAuth is the seam between this app and Google Identity Services,
// kept narrow and behind an interface specifically so
// handlers_auth_test.go can fake a full sign-in without ever making a real
// network call.
type googleAuth interface {
	// VerifyIDToken checks a signed ID-token JWT (posted straight from the
	// browser by Google's "Sign in with Google" button — see
	// web/templates/pages/login.html) and returns the identity it
	// attests to, or an error if it's invalid, expired, or wasn't issued
	// for this app.
	VerifyIDToken(ctx context.Context, idToken string) (googleIdentity, error)
}

// googleTokenInfoURL validates an ID token by asking Google directly,
// rather than this app verifying the JWT signature itself against Google's
// JWKS — Google documents this endpoint as valid for exactly this use case
// for lower-volume apps, and it means no JWT/JWKS library at all. (Google's
// own guidance is to prefer local signature verification at high request
// volume; a single self-hosted habit tracker is nowhere near that.)
const googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

type realGoogleAuth struct {
	clientID string
}

func newGoogleAuth(cfg config.Config) *realGoogleAuth {
	return &realGoogleAuth{clientID: cfg.GoogleClientID}
}

func (g *realGoogleAuth) VerifyIDToken(ctx context.Context, idToken string) (googleIdentity, error) {
	reqURL := googleTokenInfoURL + "?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("build tokeninfo request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("fetch tokeninfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return googleIdentity{}, fmt.Errorf("tokeninfo request returned status %d (token invalid or expired)", resp.StatusCode)
	}

	var body struct {
		Aud           string `json:"aud"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"` // this endpoint returns "true"/"false" as a string, not a JSON bool
		Sub           string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return googleIdentity{}, fmt.Errorf("decode tokeninfo: %w", err)
	}

	// The audience must be this app's own client id — otherwise a token
	// minted for some other Google-sign-in-using app could be replayed
	// against us by anyone who obtained one.
	if body.Aud != g.clientID {
		return googleIdentity{}, fmt.Errorf("tokeninfo audience %q does not match this app's client id", body.Aud)
	}

	return googleIdentity{Email: body.Email, EmailVerified: body.EmailVerified == "true", Sub: body.Sub}, nil
}
