package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

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

// googleAuth is the seam between this app and Google's OAuth2 flow, kept
// narrow and behind an interface specifically so handlers_auth_test.go can
// fake a full sign-in without ever making a real network call.
type googleAuth interface {
	AuthCodeURL(state string) string
	// Exchange trades an authorization code for tokens, then fetches the
	// signed-in account's identity — combined into one call because
	// nothing in this app ever needs the raw token by itself.
	Exchange(ctx context.Context, code string) (googleIdentity, error)
}

const googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

type realGoogleAuth struct {
	oauthCfg *oauth2.Config
}

func newGoogleAuth(cfg config.Config) *realGoogleAuth {
	return &realGoogleAuth{oauthCfg: &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.BaseURL + "/auth/google/callback",
		Scopes:       []string{"openid", "email"},
		Endpoint:     google.Endpoint,
	}}
}

func (g *realGoogleAuth) AuthCodeURL(state string) string {
	return g.oauthCfg.AuthCodeURL(state)
}

func (g *realGoogleAuth) Exchange(ctx context.Context, code string) (googleIdentity, error) {
	token, err := g.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("exchange code: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return googleIdentity{}, fmt.Errorf("userinfo request returned status %d", resp.StatusCode)
	}

	var body struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Sub           string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return googleIdentity{}, fmt.Errorf("decode userinfo: %w", err)
	}
	return googleIdentity{Email: body.Email, EmailVerified: body.EmailVerified, Sub: body.Sub}, nil
}
