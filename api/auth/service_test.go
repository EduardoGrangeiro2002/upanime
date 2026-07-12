package auth

import (
	"testing"
	"time"
)

func TestNeedsMFA(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-24 * time.Hour)
	stale := now.Add(-31 * 24 * time.Hour)

	cases := []struct {
		name         string
		lastIP       string
		ip           string
		lastLocation string
		location     string
		lastMFAAt    time.Time
		expected     bool
	}{
		{"never validated", "", "1.2.3.4", "", "", time.Time{}, true},
		{"validated over 30 days ago", "1.2.3.4", "1.2.3.4", "SP, Brazil", "SP, Brazil", stale, true},
		{"ip changed", "1.2.3.4", "5.6.7.8", "SP, Brazil", "SP, Brazil", recent, true},
		{"location changed", "1.2.3.4", "1.2.3.4", "SP, Brazil", "RJ, Brazil", recent, true},
		{"same context within window", "1.2.3.4", "1.2.3.4", "SP, Brazil", "SP, Brazil", recent, false},
		{"geo lookup unavailable does not trigger", "1.2.3.4", "1.2.3.4", "SP, Brazil", "", recent, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NeedsMFA(tc.lastIP, tc.ip, tc.lastLocation, tc.location, tc.lastMFAAt, now)
			if got != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestGenerateTempPasswordIsRandomAndLongEnough(t *testing.T) {
	first, err := GenerateTempPassword()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateTempPassword()
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatal("expected distinct temp passwords")
	}
	if len(first) < 12 {
		t.Fatalf("temp password too short: %d chars", len(first))
	}
}
