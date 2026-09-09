package services

import (
	"testing"
	"time"
)

func TestMarketTransition(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-09-09T13:30:30Z")
	event, at, ok := marketTransition(now, 2*time.Minute)
	if !ok || event != "OPEN" || at.Format(time.RFC3339) != "2026-09-09T13:30:00Z" {
		t.Fatalf("unexpected transition: %s %s %v", event, at, ok)
	}

	now, _ = time.Parse(time.RFC3339, "2026-09-09T20:00:30Z")
	event, _, ok = marketTransition(now, 2*time.Minute)
	if !ok || event != "CLOSED" {
		t.Fatalf("expected close transition, got %s %v", event, ok)
	}
}
