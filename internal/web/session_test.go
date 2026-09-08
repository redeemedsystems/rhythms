package web

import (
	"testing"
	"time"
)

func TestSessionSignVerifyRoundTrip(t *testing.T) {
	secret := []byte("test-secret")
	value := signSession(secret, 42, time.Now().Add(time.Hour))

	userID, ok := verifySession(secret, value)
	if !ok {
		t.Fatal("verifySession: expected ok=true for a freshly signed session")
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
}

func TestSessionVerifyRejectsTamperedPayload(t *testing.T) {
	secret := []byte("test-secret")
	value := signSession(secret, 42, time.Now().Add(time.Hour))

	tampered := "99" + value[2:] // flip the userID, keep the (now-invalid) signature
	if _, ok := verifySession(secret, tampered); ok {
		t.Error("verifySession: expected ok=false for a tampered payload")
	}
}

func TestSessionVerifyRejectsExpired(t *testing.T) {
	secret := []byte("test-secret")
	value := signSession(secret, 42, time.Now().Add(-time.Hour))

	if _, ok := verifySession(secret, value); ok {
		t.Error("verifySession: expected ok=false for an expired session")
	}
}

func TestSessionVerifyRejectsWrongSecret(t *testing.T) {
	value := signSession([]byte("secret-a"), 42, time.Now().Add(time.Hour))

	if _, ok := verifySession([]byte("secret-b"), value); ok {
		t.Error("verifySession: expected ok=false when verifying with a different secret")
	}
}

func TestSessionVerifyRejectsMalformedValue(t *testing.T) {
	secret := []byte("test-secret")
	for _, bad := range []string{"", "no-dots-at-all", "1.2", "abc.def.ghi"} {
		if _, ok := verifySession(secret, bad); ok {
			t.Errorf("verifySession(%q) = ok=true, want ok=false", bad)
		}
	}
}
