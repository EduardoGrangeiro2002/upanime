package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"upanime/api/auth"
	"upanime/api/handler"
	"upanime/api/store"
	"upanime/api/testutil"
)

func setupTokenTest(t *testing.T, token string) *chi.Mux {
	t.Helper()

	db := testutil.NewTestDB(t)
	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { redisClient.Close() })

	service := auth.NewService(
		store.NewSQLiteUserStore(db),
		auth.NewCodeStore(redisClient),
		&recordingMailer{},
		&staticGeo{location: "São Paulo, Brazil"},
		auth.NewTokenSigner("test-secret"),
		func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) },
	)

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(handler.RequireAuthOrToken(service, token))
		r.Get("/api/dataset/stats", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
	return router
}

func TestRequireAuthOrToken_ValidBearerPasses(t *testing.T) {
	router := setupTokenTest(t, "segredo-de-maquina")

	request := httptest.NewRequest("GET", "/api/dataset/stats", nil)
	request.Header.Set("Authorization", "Bearer segredo-de-maquina")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("expected 200 with valid bearer, got %d", response.Code)
	}
}

func TestRequireAuthOrToken_WrongBearerFallsToSession(t *testing.T) {
	router := setupTokenTest(t, "segredo-de-maquina")

	request := httptest.NewRequest("GET", "/api/dataset/stats", nil)
	request.Header.Set("Authorization", "Bearer errado")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong bearer and no session, got %d", response.Code)
	}
}

func TestRequireAuthOrToken_EmptyTokenDisablesBearer(t *testing.T) {
	router := setupTokenTest(t, "")

	request := httptest.NewRequest("GET", "/api/dataset/stats", nil)
	request.Header.Set("Authorization", "Bearer qualquer-coisa")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when bearer auth is disabled, got %d", response.Code)
	}
}

func TestRequireAuthOrToken_NoCredentials(t *testing.T) {
	router := setupTokenTest(t, "segredo-de-maquina")

	request := httptest.NewRequest("GET", "/api/dataset/stats", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without credentials, got %d", response.Code)
	}
}
