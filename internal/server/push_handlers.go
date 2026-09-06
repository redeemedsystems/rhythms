package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"rhythms/internal/auth"
	"rhythms/internal/push"
	"rhythms/internal/store"
)

func (s *Server) handleVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"publicKey": s.vapidPublicKey})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (s *Server) handlePushSubscribe(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := store.SaveSubscription(s.db, user.ID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth, r.UserAgent()); err != nil {
		s.internalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (s *Server) handlePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req unsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := store.DeleteSubscription(s.db, req.Endpoint); err != nil {
		s.internalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDebugTestPush sends a one-off test notification to the current
// user's push subscriptions. Only registered when config.Debug is set.
func (s *Server) handleDebugTestPush(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	subs, err := store.ListSubscriptionsForUser(s.db, user.ID)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if len(subs) == 0 {
		http.Error(w, "no push subscriptions for this user", http.StatusBadRequest)
		return
	}

	payload := push.Payload{Title: "Rhythms (test)", Body: "This is a test notification.", URL: "/today"}
	for _, sub := range subs {
		if err := s.sender.Send(sub, payload); err != nil {
			slog.Warn("debug test push failed", "err", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
