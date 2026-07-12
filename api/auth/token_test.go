package auth

import (
	"testing"
	"time"
)

func TestTokenSignAndVerify(t *testing.T) {
	signer := NewTokenSigner("test-secret")
	now := time.Now()

	token := signer.Sign("user@example.com", now.Add(time.Hour))

	email, valid := signer.Verify(token, now)
	if !valid {
		t.Fatal("expected token to be valid")
	}
	if email != "user@example.com" {
		t.Fatalf("unexpected email: %s", email)
	}
}

func TestTokenExpired(t *testing.T) {
	signer := NewTokenSigner("test-secret")
	now := time.Now()

	token := signer.Sign("user@example.com", now.Add(-time.Minute))

	if _, valid := signer.Verify(token, now); valid {
		t.Fatal("expected expired token to be invalid")
	}
}

func TestTokenTampered(t *testing.T) {
	signer := NewTokenSigner("test-secret")
	token := signer.Sign("user@example.com", time.Now().Add(time.Hour))

	if _, valid := signer.Verify(token+"x", time.Now()); valid {
		t.Fatal("expected tampered token to be invalid")
	}
}

func TestTokenWrongSecret(t *testing.T) {
	token := NewTokenSigner("secret-a").Sign("user@example.com", time.Now().Add(time.Hour))

	if _, valid := NewTokenSigner("secret-b").Verify(token, time.Now()); valid {
		t.Fatal("expected token signed with another secret to be invalid")
	}
}

func TestTokenGarbage(t *testing.T) {
	signer := NewTokenSigner("test-secret")
	for _, garbage := range []string{"", "abc", "a.b", "!!!.???"} {
		if _, valid := signer.Verify(garbage, time.Now()); valid {
			t.Fatalf("expected garbage token %q to be invalid", garbage)
		}
	}
}
