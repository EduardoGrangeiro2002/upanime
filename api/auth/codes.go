package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	PurposeMFA   = "mfa"
	PurposeReset = "reset"

	codeTTL         = 15 * time.Minute
	maxAttempts     = 5
	issueCooldown   = time.Minute
	rateLimitWindow = 15 * time.Minute
	rateLimitMax    = 30
)

var ErrCooldown = errors.New("aguarde antes de pedir um novo código")

type CodeStore struct {
	client *redis.Client
}

func NewCodeStore(client *redis.Client) *CodeStore {
	return &CodeStore{client: client}
}

func (c *CodeStore) Issue(ctx context.Context, purpose, email string) (string, error) {
	acquired, err := c.client.SetNX(ctx, cooldownKey(purpose, email), "1", issueCooldown).Result()
	if err != nil {
		return "", err
	}
	if !acquired {
		return "", ErrCooldown
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())

	pipe := c.client.TxPipeline()
	pipe.Set(ctx, codeKey(purpose, email), code, codeTTL)
	pipe.Del(ctx, attemptsKey(purpose, email))
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return code, nil
}

func (c *CodeStore) Verify(ctx context.Context, purpose, email, code string) (bool, error) {
	stored, err := c.client.Get(ctx, codeKey(purpose, email)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if subtle.ConstantTimeCompare([]byte(stored), []byte(code)) == 1 {
		c.client.Del(ctx, codeKey(purpose, email), attemptsKey(purpose, email))
		return true, nil
	}

	attempts, err := c.client.Incr(ctx, attemptsKey(purpose, email)).Result()
	if err != nil {
		return false, err
	}
	c.client.Expire(ctx, attemptsKey(purpose, email), codeTTL)
	if attempts >= maxAttempts {
		c.client.Del(ctx, codeKey(purpose, email), attemptsKey(purpose, email))
	}
	return false, nil
}

func (c *CodeStore) AllowIP(ctx context.Context, ip string) (bool, error) {
	key := "auth:rl:" + ip
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		c.client.Expire(ctx, key, rateLimitWindow)
	}
	return count <= rateLimitMax, nil
}

func codeKey(purpose, email string) string {
	return "auth:" + purpose + ":" + email
}

func attemptsKey(purpose, email string) string {
	return "auth:" + purpose + ":" + email + ":attempts"
}

func cooldownKey(purpose, email string) string {
	return "auth:" + purpose + ":" + email + ":cooldown"
}
