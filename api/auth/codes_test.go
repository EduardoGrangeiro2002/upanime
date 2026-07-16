package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCodeStore(t *testing.T) (*CodeStore, *miniredis.Miniredis) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { client.Close() })
	return NewCodeStore(client), mini
}

func TestIssueEnforcesCooldown(t *testing.T) {
	codes, mini := newTestCodeStore(t)

	code, err := codes.Issue(t.Context(), PurposeMFA, "dono@upanime.dev")
	if err != nil || code == "" {
		t.Fatalf("first issue failed: %v", err)
	}

	if _, err := codes.Issue(t.Context(), PurposeMFA, "dono@upanime.dev"); !errors.Is(err, ErrCooldown) {
		t.Fatalf("expected ErrCooldown, got %v", err)
	}

	valid, err := codes.Verify(t.Context(), PurposeMFA, "dono@upanime.dev", code)
	if err != nil || !valid {
		t.Fatalf("code should stay valid during cooldown: valid=%v err=%v", valid, err)
	}

	mini.FastForward(issueCooldown + time.Second)
	if _, err := codes.Issue(t.Context(), PurposeMFA, "dono@upanime.dev"); err != nil {
		t.Fatalf("issue after cooldown failed: %v", err)
	}
}

func TestCooldownScopedByPurposeAndEmail(t *testing.T) {
	codes, _ := newTestCodeStore(t)

	if _, err := codes.Issue(t.Context(), PurposeMFA, "dono@upanime.dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := codes.Issue(t.Context(), PurposeReset, "dono@upanime.dev"); err != nil {
		t.Fatalf("reset purpose should not share cooldown: %v", err)
	}
	if _, err := codes.Issue(t.Context(), PurposeMFA, "outra@upanime.dev"); err != nil {
		t.Fatalf("other email should not share cooldown: %v", err)
	}
}

func TestCooldownDoesNotResetAttempts(t *testing.T) {
	codes, _ := newTestCodeStore(t)

	code, err := codes.Issue(t.Context(), PurposeMFA, "dono@upanime.dev")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < maxAttempts-1; i++ {
		codes.Verify(t.Context(), PurposeMFA, "dono@upanime.dev", "000000")
	}

	if _, err := codes.Issue(t.Context(), PurposeMFA, "dono@upanime.dev"); !errors.Is(err, ErrCooldown) {
		t.Fatalf("expected ErrCooldown, got %v", err)
	}

	codes.Verify(t.Context(), PurposeMFA, "dono@upanime.dev", "000000")

	valid, err := codes.Verify(t.Context(), PurposeMFA, "dono@upanime.dev", code)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("code should be invalidated after max attempts even with re-issue in between")
	}
}

func TestAllowIPLimitsAndResets(t *testing.T) {
	codes, mini := newTestCodeStore(t)

	for i := 0; i < rateLimitMax; i++ {
		allowed, err := codes.AllowIP(t.Context(), "203.0.113.9")
		if err != nil || !allowed {
			t.Fatalf("request %d should be allowed: %v", i, err)
		}
	}

	allowed, err := codes.AllowIP(t.Context(), "203.0.113.9")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("request beyond limit should be denied")
	}

	allowed, err = codes.AllowIP(t.Context(), "198.51.100.1")
	if err != nil || !allowed {
		t.Fatalf("other ip should not be affected: %v", err)
	}

	mini.FastForward(rateLimitWindow + time.Second)
	allowed, err = codes.AllowIP(t.Context(), "203.0.113.9")
	if err != nil || !allowed {
		t.Fatalf("limit should reset after window: %v", err)
	}
}
