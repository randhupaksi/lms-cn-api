package auth

import (
	"testing"
	"time"
)

func TestLoginGuardBlocksAndResetsIdentifier(t *testing.T) {
	now := time.Date(2026, time.September, 2, 10, 0, 0, 0, time.UTC)
	guard := NewLoginGuard(2, time.Minute)
	guard.now = func() time.Time { return now }

	if !guard.Consume(" Student-01 ") || !guard.Consume("student-01") {
		t.Fatal("expected attempts below the limit to be accepted")
	}
	if guard.Consume("STUDENT-01") {
		t.Fatal("expected normalized identifier to be blocked at the limit")
	}

	guard.Reset("student-01")
	if !guard.Consume("student-01") {
		t.Fatal("expected a successful-login reset to clear the bucket")
	}
}

func TestLoginGuardReopensAfterWindow(t *testing.T) {
	now := time.Date(2026, time.September, 2, 10, 0, 0, 0, time.UTC)
	guard := NewLoginGuard(1, time.Minute)
	guard.now = func() time.Time { return now }

	if !guard.Consume("student-01") || guard.Consume("student-01") {
		t.Fatal("expected the second attempt to be blocked")
	}
	now = now.Add(time.Minute)
	if !guard.Consume("student-01") {
		t.Fatal("expected the bucket to reopen after its window")
	}
}
