package auth

import (
	"testing"
	"time"

	"lms-cn-api/internal/authz"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	now := time.Date(2026, time.September, 2, 8, 0, 0, 0, time.UTC)
	manager := NewTokenManager("a-secret-that-is-long-enough-for-tests", "citra-negara-lms", 15*time.Minute)
	want := authz.Principal{UserID: "user-1", SessionID: "session-1", Role: "student", MustChangePassword: true}

	raw, err := manager.CreateAccessToken(want, now)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	got, err := manager.VerifyAccessToken(raw)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if got != want {
		t.Fatalf("VerifyAccessToken() = %#v, want %#v", got, want)
	}
}

func TestAccessTokenRejectsDifferentIssuer(t *testing.T) {
	now := time.Now().UTC()
	issuerA := NewTokenManager("a-secret-that-is-long-enough-for-tests", "issuer-a", 15*time.Minute)
	issuerB := NewTokenManager("a-secret-that-is-long-enough-for-tests", "issuer-b", 15*time.Minute)
	raw, err := issuerA.CreateAccessToken(authz.Principal{UserID: "user-1", SessionID: "session-1", Role: "teacher"}, now)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	if _, err := issuerB.VerifyAccessToken(raw); err == nil {
		t.Fatal("VerifyAccessToken() expected issuer validation error")
	}
}
