package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

type progressEnv struct {
	router *chi.Mux
	anime  *model.Anime
}

func setupProgressHandler(t *testing.T) *progressEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	anime := &model.Anime{
		Title:     "Handler Progress Anime",
		URL:       "https://animesonlinecc.to/anime/handler-progress",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/hp-1", Type: "episode"},
				{Title: "Ep 2", Number: "2", URL: "https://example.com/hp-2", Type: "episode"},
			}},
		},
	}
	if err := animes.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	h := NewProgressHandler(store.NewSQLiteWatchProgressStore(db))
	router := chi.NewRouter()
	router.Get("/api/progress", h.List)
	router.Get("/api/progress/episode/{id}", h.Get)
	router.Put("/api/progress/episode/{id}", h.Update)

	return &progressEnv{router: router, anime: anime}
}

func (e *progressEnv) do(t *testing.T, method, path, body, email string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), userEmailKey, email))
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func TestProgressHandler_UpdateAndGet(t *testing.T) {
	env := setupProgressHandler(t)
	epID := env.anime.Seasons[0].Episodes[0].ID.Int64()

	w := env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", epID), `{"position": 123.4, "duration": 1420}`, "eu@teste.dev")
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	w = env.do(t, "GET", fmt.Sprintf("/api/progress/episode/%d", epID), "", "eu@teste.dev")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var p model.WatchProgress
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.Position != 123.4 || p.Duration != 1420 {
		t.Fatalf("expected 123.4/1420, got %v/%v", p.Position, p.Duration)
	}
	if p.UpdatedAt == "" {
		t.Fatal("expected updatedAt set")
	}
}

func TestProgressHandler_GetOtherUserIsMissing(t *testing.T) {
	env := setupProgressHandler(t)
	epID := env.anime.Seasons[0].Episodes[0].ID.Int64()

	env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", epID), `{"position": 100, "duration": 1400}`, "eu@teste.dev")

	w := env.do(t, "GET", fmt.Sprintf("/api/progress/episode/%d", epID), "", "outra@teste.dev")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for another user, got %d", w.Code)
	}
}

func TestProgressHandler_GetMissing(t *testing.T) {
	env := setupProgressHandler(t)

	w := env.do(t, "GET", "/api/progress/episode/9999", "", "eu@teste.dev")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestProgressHandler_UpdateUnknownEpisode(t *testing.T) {
	env := setupProgressHandler(t)

	w := env.do(t, "PUT", "/api/progress/episode/9999", `{"position": 10, "duration": 100}`, "eu@teste.dev")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProgressHandler_UpdateInvalidBody(t *testing.T) {
	env := setupProgressHandler(t)
	epID := env.anime.Seasons[0].Episodes[0].ID.Int64()

	w := env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", epID), `{invalid`, "eu@teste.dev")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestProgressHandler_UpdateNegativePosition(t *testing.T) {
	env := setupProgressHandler(t)
	epID := env.anime.Seasons[0].Episodes[0].ID.Int64()

	w := env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", epID), `{"position": -1, "duration": 100}`, "eu@teste.dev")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestProgressHandler_UpdateInvalidID(t *testing.T) {
	env := setupProgressHandler(t)

	w := env.do(t, "PUT", "/api/progress/episode/abc", `{"position": 10, "duration": 100}`, "eu@teste.dev")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestProgressHandler_List(t *testing.T) {
	env := setupProgressHandler(t)
	eps := env.anime.Seasons[0].Episodes

	env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", eps[0].ID.Int64()), `{"position": 100, "duration": 1400}`, "eu@teste.dev")
	env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", eps[1].ID.Int64()), `{"position": 1390, "duration": 1400}`, "eu@teste.dev")
	env.do(t, "PUT", fmt.Sprintf("/api/progress/episode/%d", eps[1].ID.Int64()), `{"position": 300, "duration": 1400}`, "outra@teste.dev")

	w := env.do(t, "GET", "/api/progress", "", "eu@teste.dev")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var items []model.WatchProgress
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (completed excluded, other user excluded), got %d", len(items))
	}
	if items[0].AnimeTitle != "Handler Progress Anime" {
		t.Errorf("expected anime title joined, got '%s'", items[0].AnimeTitle)
	}
	if items[0].EpisodeID != eps[0].ID {
		t.Errorf("expected episode %v, got %v", eps[0].ID, items[0].EpisodeID)
	}
}

func TestProgressHandler_ListEmpty(t *testing.T) {
	env := setupProgressHandler(t)

	w := env.do(t, "GET", "/api/progress", "", "eu@teste.dev")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); body != "[]" {
		t.Fatalf("expected empty array, got %s", body)
	}
}
