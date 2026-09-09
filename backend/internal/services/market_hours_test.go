package services

import (
	"testing"
	"time"
)

func TestUSRegularMarketStatus(t *testing.T) {
	tests := []struct{ name, now, status, until string }{
		{"open", "2026-09-09T15:00:00Z", "OPEN", "2026-09-09T20:00:00Z"},
		{"before open", "2026-09-09T12:00:00Z", "CLOSED", "2026-09-09T13:30:00Z"},
		{"weekend", "2026-09-12T15:00:00Z", "CLOSED", "2026-09-14T13:30:00Z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, test.now)
			status, until := usRegularMarketStatus(now)
			if status != test.status || until.Format(time.RFC3339) != test.until {
				t.Fatalf("got %s until %s", status, until.Format(time.RFC3339))
			}
		})
	}
}

func TestUSRegularMarketStatusSkipsHoliday(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-12-25T15:00:00Z")
	status, until := usRegularMarketStatus(now)
	if status != "CLOSED" || until.Format(time.RFC3339) != "2026-12-28T14:30:00Z" {
		t.Fatalf("expected Christmas closure until Monday, got %s %s", status, until)
	}
}
