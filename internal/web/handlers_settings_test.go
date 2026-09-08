package web

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleSettingsShowsPushToggle(t *testing.T) {
	s, _, _ := newTestServer(t)

	rec := doRequest(t, s, http.MethodGet, "/settings", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="push-toggle"`) {
		t.Errorf("settings page missing push-toggle button: %s", rec.Body.String())
	}
}

func TestHandleTodayNoLongerShowsPushToggle(t *testing.T) {
	s, _, _ := newTestServer(t)

	rec := doRequest(t, s, http.MethodGet, "/today", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `id="push-toggle"`) {
		t.Errorf("push-toggle should have moved off /today: %s", rec.Body.String())
	}
}
