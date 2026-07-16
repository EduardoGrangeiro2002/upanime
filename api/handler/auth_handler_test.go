package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"upanime/api/auth"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

type recordingMailer struct {
	emails []string
	bodies []string
}

func (m *recordingMailer) Send(to, subject, body string) error {
	m.emails = append(m.emails, to)
	m.bodies = append(m.bodies, body)
	return nil
}

type staticGeo struct {
	location string
}

func (g *staticGeo) Lookup(_ context.Context, _ string) string {
	return g.location
}

type authTestEnv struct {
	router *chi.Mux
	users  *store.SQLiteUserStore
	redis  *miniredis.Miniredis
	mailer *recordingMailer
	geo    *staticGeo
	now    time.Time
}

func setupAuthTest(t *testing.T) *authTestEnv {
	t.Helper()

	db := testutil.NewTestDB(t)
	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { redisClient.Close() })

	env := &authTestEnv{
		users:  store.NewSQLiteUserStore(db),
		redis:  mini,
		mailer: &recordingMailer{},
		geo:    &staticGeo{location: "São Paulo, Brazil"},
		now:    time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
	}

	codeStore := auth.NewCodeStore(redisClient)
	service := auth.NewService(
		env.users,
		codeStore,
		env.mailer,
		env.geo,
		auth.NewTokenSigner("test-secret"),
		func() time.Time { return env.now },
	)
	authHandler := handler.NewAuthHandler(service, env.users, false)
	inviteHandler := handler.NewInviteHandler(env.users, env.mailer)

	router := chi.NewRouter()
	router.Route("/api/auth", func(r chi.Router) {
		r.Use(handler.RateLimitAuth(codeStore))
		r.Post("/login", authHandler.Login)
		r.Post("/change-password", authHandler.ChangePassword)
		r.Post("/mfa", authHandler.VerifyMFA)
		r.Post("/forgot", authHandler.Forgot)
		r.Post("/reset", authHandler.Reset)
		r.Post("/logout", authHandler.Logout)
	})
	router.Group(func(r chi.Router) {
		r.Use(handler.RequireAuth(service))
		r.Get("/api/auth/me", authHandler.Me)
		r.Get("/api/protected", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		r.Group(func(a chi.Router) {
			a.Use(handler.RequireAdmin(env.users))
			a.Post("/api/invites", inviteHandler.Create)
			a.Get("/api/users", inviteHandler.ListUsers)
		})
	})
	env.router = router
	return env
}

func (e *authTestEnv) createUser(t *testing.T, email, password string, mustChange bool) {
	t.Helper()
	e.createUserWithRole(t, email, password, mustChange, false)
}

func (e *authTestEnv) createUserWithRole(t *testing.T, email, password string, mustChange, isAdmin bool) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{Email: email, PasswordHash: hash, MustChangePassword: mustChange, IsAdmin: isAdmin}
	if err := e.users.Create(t.Context(), user); err != nil {
		t.Fatal(err)
	}
}

func (e *authTestEnv) sessionFor(t *testing.T, email, password string) *http.Cookie {
	t.Helper()
	e.users.UpdateMFAContext(t.Context(), email, "203.0.113.10", "São Paulo, Brazil", e.now.Add(-time.Hour))
	login := e.post(t, "/api/auth/login", map[string]string{"email": email, "password": password})
	cookie := sessionCookie(login)
	if cookie == nil {
		t.Fatalf("expected direct session for %s, got %d %s", email, login.Code, login.Body.String())
	}
	return cookie
}

func (e *authTestEnv) post(t *testing.T, path string, payload map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest("POST", path, bytes.NewReader(body))
	request.RemoteAddr = "203.0.113.10:51000"
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	e.router.ServeHTTP(response, request)
	return response
}

func (e *authTestEnv) storedCode(t *testing.T, purpose, email string) string {
	t.Helper()
	code, err := e.redis.Get(fmt.Sprintf("auth:%s:%s", purpose, email))
	if err != nil {
		t.Fatalf("code not found in redis: %v", err)
	}
	return code
}

func decodeStep(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload["step"]
}

func sessionCookie(response *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "upanime_session" && cookie.Value != "" {
			return cookie
		}
	}
	return nil
}

func TestFirstLoginForcesPasswordChangeThenMFA(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-temporaria", true)

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-temporaria",
	})
	if login.Code != http.StatusOK || decodeStep(t, login) != "change_password" {
		t.Fatalf("expected change_password step, got %d %s", login.Code, login.Body.String())
	}

	change := env.post(t, "/api/auth/change-password", map[string]string{
		"email": "dono@upanime.dev", "currentPassword": "senha-temporaria", "newPassword": "senha-definitiva",
	})
	if change.Code != http.StatusOK || decodeStep(t, change) != "mfa" {
		t.Fatalf("expected mfa step after password change, got %d %s", change.Code, change.Body.String())
	}
	if len(env.mailer.emails) != 1 || env.mailer.emails[0] != "dono@upanime.dev" {
		t.Fatalf("expected mfa email sent, got %v", env.mailer.emails)
	}

	code := env.storedCode(t, "mfa", "dono@upanime.dev")
	verify := env.post(t, "/api/auth/mfa", map[string]string{
		"email": "dono@upanime.dev", "code": code,
	})
	if verify.Code != http.StatusOK || decodeStep(t, verify) != "ok" {
		t.Fatalf("expected ok step, got %d %s", verify.Code, verify.Body.String())
	}

	cookie := sessionCookie(verify)
	if cookie == nil {
		t.Fatal("expected session cookie after mfa")
	}
	if !cookie.HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}

	request := httptest.NewRequest("GET", "/api/protected", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	env.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected protected route to accept session, got %d", response.Code)
	}
}

func TestLoginSkipsMFAWithinWindowAndSameContext(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)
	env.users.UpdateMFAContext(t.Context(), "dono@upanime.dev", "203.0.113.10", "São Paulo, Brazil", env.now.Add(-24*time.Hour))

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})

	if decodeStep(t, login) != "ok" {
		t.Fatalf("expected direct ok, got %s", login.Body.String())
	}
	if sessionCookie(login) == nil {
		t.Fatal("expected session cookie on direct login")
	}
}

func TestLoginRequiresMFAWhenIPChanges(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)
	env.users.UpdateMFAContext(t.Context(), "dono@upanime.dev", "198.51.100.7", "São Paulo, Brazil", env.now.Add(-24*time.Hour))

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})

	if decodeStep(t, login) != "mfa" {
		t.Fatalf("expected mfa on ip change, got %s", login.Body.String())
	}
}

func TestLoginRequiresMFAWhenLocationChanges(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)
	env.users.UpdateMFAContext(t.Context(), "dono@upanime.dev", "203.0.113.10", "Lisboa, Portugal", env.now.Add(-24*time.Hour))

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})

	if decodeStep(t, login) != "mfa" {
		t.Fatalf("expected mfa on location change, got %s", login.Body.String())
	}
}

func TestLoginRequiresMFAAfterThirtyDays(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)
	env.users.UpdateMFAContext(t.Context(), "dono@upanime.dev", "203.0.113.10", "São Paulo, Brazil", env.now.Add(-31*24*time.Hour))

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})

	if decodeStep(t, login) != "mfa" {
		t.Fatalf("expected mfa after 30 days, got %s", login.Body.String())
	}
}

func TestWrongPasswordReturns401(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "errada",
	})
	if login.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", login.Code)
	}

	unknown := env.post(t, "/api/auth/login", map[string]string{
		"email": "naoexiste@upanime.dev", "password": "qualquer",
	})
	if unknown.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown user, got %d", unknown.Code)
	}
}

func TestMFACodeBruteForceInvalidatesCode(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)

	env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})
	realCode := env.storedCode(t, "mfa", "dono@upanime.dev")

	for i := 0; i < 5; i++ {
		response := env.post(t, "/api/auth/mfa", map[string]string{
			"email": "dono@upanime.dev", "code": "000000",
		})
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i, response.Code)
		}
	}

	response := env.post(t, "/api/auth/mfa", map[string]string{
		"email": "dono@upanime.dev", "code": realCode,
	})
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected real code to be invalidated after brute force, got %d", response.Code)
	}
}

func TestForgotAndResetPassword(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-antiga", false)

	forgot := env.post(t, "/api/auth/forgot", map[string]string{"email": "dono@upanime.dev"})
	if forgot.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", forgot.Code)
	}

	code := env.storedCode(t, "reset", "dono@upanime.dev")
	reset := env.post(t, "/api/auth/reset", map[string]string{
		"email": "dono@upanime.dev", "code": code, "newPassword": "senha-nova-123",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", reset.Code, reset.Body.String())
	}

	oldLogin := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-antiga",
	})
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password to be rejected, got %d", oldLogin.Code)
	}

	newLogin := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-nova-123",
	})
	if newLogin.Code != http.StatusOK {
		t.Fatalf("expected new password to work, got %d", newLogin.Code)
	}
}

func TestForgotUnknownEmailDoesNotLeak(t *testing.T) {
	env := setupAuthTest(t)

	forgot := env.post(t, "/api/auth/forgot", map[string]string{"email": "fantasma@upanime.dev"})
	if forgot.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown email, got %d", forgot.Code)
	}
	if len(env.mailer.emails) != 0 {
		t.Fatal("expected no email sent for unknown user")
	}
}

func TestForgotCooldownLimitsEmails(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)

	first := env.post(t, "/api/auth/forgot", map[string]string{"email": "dono@upanime.dev"})
	second := env.post(t, "/api/auth/forgot", map[string]string{"email": "dono@upanime.dev"})
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected 200 on both requests, got %d and %d", first.Code, second.Code)
	}
	if len(env.mailer.emails) != 1 {
		t.Fatalf("expected 1 email during cooldown, got %d", len(env.mailer.emails))
	}

	code := env.storedCode(t, "reset", "dono@upanime.dev")
	reset := env.post(t, "/api/auth/reset", map[string]string{
		"email": "dono@upanime.dev", "code": code, "newPassword": "senha-nova-123",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("original code should stay valid during cooldown, got %d", reset.Code)
	}

	env.redis.FastForward(61 * time.Second)
	env.post(t, "/api/auth/forgot", map[string]string{"email": "dono@upanime.dev"})
	if len(env.mailer.emails) != 2 {
		t.Fatalf("expected new email after cooldown, got %d", len(env.mailer.emails))
	}
}

func TestLoginDuringCooldownKeepsCodeAndSkipsResend(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-definitiva", false)

	env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})
	code := env.storedCode(t, "mfa", "dono@upanime.dev")

	retry := env.post(t, "/api/auth/login", map[string]string{
		"email": "dono@upanime.dev", "password": "senha-definitiva",
	})
	if retry.Code != http.StatusOK || decodeStep(t, retry) != "mfa" {
		t.Fatalf("expected mfa step on retry, got %d %s", retry.Code, retry.Body.String())
	}
	if len(env.mailer.emails) != 1 {
		t.Fatalf("expected single email during cooldown, got %d", len(env.mailer.emails))
	}

	verify := env.post(t, "/api/auth/mfa", map[string]string{
		"email": "dono@upanime.dev", "code": code,
	})
	if verify.Code != http.StatusOK || decodeStep(t, verify) != "ok" {
		t.Fatalf("expected original code to work, got %d %s", verify.Code, verify.Body.String())
	}
}

func TestAuthRateLimitByIP(t *testing.T) {
	env := setupAuthTest(t)

	for i := 0; i < 30; i++ {
		response := env.post(t, "/api/auth/login", map[string]string{
			"email": "x@upanime.dev", "password": "errada",
		})
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: expected 401, got %d", i, response.Code)
		}
	}

	blocked := env.post(t, "/api/auth/login", map[string]string{
		"email": "x@upanime.dev", "password": "errada",
	})
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit, got %d", blocked.Code)
	}

	env.redis.FastForward(16 * time.Minute)
	after := env.post(t, "/api/auth/login", map[string]string{
		"email": "x@upanime.dev", "password": "errada",
	})
	if after.Code != http.StatusUnauthorized {
		t.Fatalf("expected limit reset after window, got %d", after.Code)
	}
}

func TestResetWithWrongCodeFails(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-antiga", false)
	env.post(t, "/api/auth/forgot", map[string]string{"email": "dono@upanime.dev"})

	reset := env.post(t, "/api/auth/reset", map[string]string{
		"email": "dono@upanime.dev", "code": "999999", "newPassword": "senha-nova-123",
	})
	if reset.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong reset code, got %d", reset.Code)
	}
}

func TestWeakNewPasswordRejected(t *testing.T) {
	env := setupAuthTest(t)
	env.createUser(t, "dono@upanime.dev", "senha-temporaria", true)

	change := env.post(t, "/api/auth/change-password", map[string]string{
		"email": "dono@upanime.dev", "currentPassword": "senha-temporaria", "newPassword": "curta",
	})
	if change.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for weak password, got %d", change.Code)
	}
}

func TestProtectedRoutesRejectWithoutSession(t *testing.T) {
	env := setupAuthTest(t)

	request := httptest.NewRequest("GET", "/api/protected", nil)
	response := httptest.NewRecorder()
	env.router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cookie, got %d", response.Code)
	}

	request = httptest.NewRequest("GET", "/api/protected", nil)
	request.AddCookie(&http.Cookie{Name: "upanime_session", Value: "token-falso"})
	response = httptest.NewRecorder()
	env.router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", response.Code)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	env := setupAuthTest(t)

	logout := env.post(t, "/api/auth/logout", map[string]string{})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", logout.Code)
	}

	for _, cookie := range logout.Result().Cookies() {
		if cookie.Name == "upanime_session" && cookie.MaxAge >= 0 {
			t.Fatal("expected session cookie to be expired")
		}
	}
}
