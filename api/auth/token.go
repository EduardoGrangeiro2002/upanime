package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TokenSigner struct {
	secret []byte
}

func NewTokenSigner(secret string) *TokenSigner {
	return &TokenSigner{secret: []byte(secret)}
}

func (t *TokenSigner) Sign(email string, expiresAt time.Time) string {
	payload := fmt.Sprintf("%s|%d", email, expiresAt.Unix())
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + t.signature(payload)
}

func (t *TokenSigner) Verify(token string, now time.Time) (string, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	payload := string(raw)

	if !hmac.Equal([]byte(t.signature(payload)), []byte(parts[1])) {
		return "", false
	}

	separator := strings.LastIndex(payload, "|")
	if separator < 0 {
		return "", false
	}
	email := payload[:separator]
	expiresUnix, err := strconv.ParseInt(payload[separator+1:], 10, 64)
	if err != nil {
		return "", false
	}
	if now.After(time.Unix(expiresUnix, 0)) {
		return "", false
	}
	return email, true
}

func (t *TokenSigner) signature(payload string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
