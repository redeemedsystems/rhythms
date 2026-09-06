package web

import (
	"encoding/json"
	"net/http"

	webassets "rhythms/web"

	"rhythms/internal/domain"
)

// handleManifest serves the PWA manifest. start_url is /today — this is
// the explicit widget replacement called out in the rewrite plan: no
// Android home-screen widget equivalent exists on the web, so installing
// the app to a home screen with /today as its landing page is the
// practical substitute.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	manifest := map[string]any{
		"name":             "Rhythms",
		"short_name":       "Rhythms",
		"description":      "A self-hosted habit tracker.",
		"start_url":        "/today",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#121212",
		"theme_color":      "#00897B",
		"icons": []map[string]any{
			{"src": "/static/icons/icon-192.png", "sizes": "192x192", "type": "image/png"},
			{"src": "/static/icons/icon-512.png", "sizes": "512x512", "type": "image/png"},
		},
	}
	w.Header().Set("Content-Type", "application/manifest+json")
	json.NewEncoder(w).Encode(manifest)
}

// handleServiceWorker serves sw.js at the root path (rather than under
// /static/, where it's also embedded) so its default scope covers the
// whole app — a service worker's scope is its own directory unless served
// from higher up, and full-app control is what lets it intercept
// navigation for every page, not just /static/.
func (s *Server) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	b, err := webassets.StaticFS.ReadFile("static/sw.js")
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(b)
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	var req pushSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	sub := domain.PushSubscription{Endpoint: req.Endpoint, P256dh: req.Keys.P256dh, Auth: req.Keys.Auth}
	if err := s.pushSubs.Upsert(r.Context(), sub); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type pushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (s *Server) handlePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req pushUnsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.pushSubs.Delete(r.Context(), req.Endpoint); err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
