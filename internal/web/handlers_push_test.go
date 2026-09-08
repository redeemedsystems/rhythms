package web

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleManifest(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/manifest.webmanifest", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Errorf("Content-Type = %q, want application/manifest+json", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{`"start_url":"/today"`, `"display":"standalone"`, "icon-192.png"} {
		if !strings.Contains(body, want) {
			t.Errorf("manifest missing %q: %s", want, body)
		}
	}
}

func TestHandleServiceWorker(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/sw.js", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "addEventListener('push'") {
		t.Errorf("expected sw.js content, got: %s", rec.Body.String())
	}
}

func TestHandlePushSubscribeAndUnsubscribe(t *testing.T) {
	s, _, _ := newTestServer(t)

	body := `{"endpoint":"https://push.example/abc","keys":{"p256dh":"pkey","auth":"akey"}}`
	rec := doRequest(t, s, http.MethodPost, "/push/subscribe", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("subscribe status = %d, want 200", rec.Code)
	}

	subs, err := s.pushSubs.List(t.Context(), testUserID)
	if err != nil || len(subs) != 1 || subs[0].Endpoint != "https://push.example/abc" {
		t.Fatalf("subs after subscribe = %+v, err=%v", subs, err)
	}

	rec = doRequest(t, s, http.MethodPost, "/push/unsubscribe", `{"endpoint":"https://push.example/abc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("unsubscribe status = %d, want 200", rec.Code)
	}
	subs, err = s.pushSubs.List(t.Context(), testUserID)
	if err != nil || len(subs) != 0 {
		t.Fatalf("subs after unsubscribe = %+v, err=%v, want empty", subs, err)
	}
}

func TestHandlePushSubscribeInvalidBody(t *testing.T) {
	s, _, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodPost, "/push/subscribe", `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
