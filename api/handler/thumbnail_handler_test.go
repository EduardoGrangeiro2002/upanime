package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

type thumbnailTestEnv struct {
	router   *chi.Mux
	animes   *store.SQLiteAnimeStore
	episodes *store.SQLiteEpisodeStore
	storage  storage.FileStorage
	calls    *atomic.Int32
}

func setupThumbnailTest(t *testing.T) *thumbnailTestEnv {
	t.Helper()

	db := testutil.NewTestDB(t)
	fs := storage.NewLocalStorage(t.TempDir())

	var calls atomic.Int32
	extractor := func(_ context.Context, _ string) ([]byte, error) {
		calls.Add(1)
		return []byte{0xFF, 0xD8, 0xFF, 0xE0, 't', 'e', 's', 't'}, nil
	}
	thumbs := service.NewThumbnailService(fs, extractor)
	episodeStore := store.NewSQLiteEpisodeStore(db)
	thumbHandler := handler.NewThumbnailHandler(episodeStore, thumbs, fs)

	router := chi.NewRouter()
	router.Get("/api/catalog/episode/{id}/thumbnail", thumbHandler.Get)

	return &thumbnailTestEnv{
		router:   router,
		animes:   store.NewSQLiteAnimeStore(db),
		episodes: episodeStore,
		storage:  fs,
		calls:    &calls,
	}
}

func (e *thumbnailTestEnv) createEpisode(t *testing.T, withVideo bool) int64 {
	t.Helper()

	anime := &model.Anime{
		Title:     "Thumb Anime",
		URL:       "https://animesonlinecc.to/anime/thumb-anime",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Label: "Season 1", Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/thumb-ep1", Type: "episode"},
			}},
		},
	}
	if err := e.animes.Create(t.Context(), anime); err != nil {
		t.Fatal(err)
	}
	episodeID := anime.Seasons[0].Episodes[0].ID.Int64()

	if !withVideo {
		return episodeID
	}

	key := "animes/thumb_anime/S1E1/ep_1.mp4"
	if err := e.storage.Save(t.Context(), key, strings.NewReader("fake-video-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := e.episodes.UpdateStorageKey(t.Context(), episodeID, key); err != nil {
		t.Fatal(err)
	}
	return episodeID
}

func (e *thumbnailTestEnv) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	e.router.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
	return response
}

func TestThumbnailGeneratedServedAndCached(t *testing.T) {
	env := setupThumbnailTest(t)
	env.createEpisode(t, true)

	first := env.get(t, "/api/catalog/episode/1/thumbnail")
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", first.Code, first.Body.String())
	}
	if contentType := first.Header().Get("Content-Type"); contentType != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", contentType)
	}
	if body := first.Body.Bytes(); len(body) < 2 || body[0] != 0xFF || body[1] != 0xD8 {
		t.Fatal("expected JPEG bytes in response")
	}
	if cache := first.Header().Get("Cache-Control"); !strings.Contains(cache, "max-age") {
		t.Fatalf("expected cache header, got %q", cache)
	}

	second := env.get(t, "/api/catalog/episode/1/thumbnail")
	if second.Code != http.StatusOK {
		t.Fatalf("expected 200 on cached request, got %d", second.Code)
	}
	if env.calls.Load() != 1 {
		t.Fatalf("expected 1 extraction across requests, got %d", env.calls.Load())
	}
}

func TestThumbnailNotDownloadedReturns404(t *testing.T) {
	env := setupThumbnailTest(t)
	env.createEpisode(t, false)

	response := env.get(t, "/api/catalog/episode/1/thumbnail")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
	if env.calls.Load() != 0 {
		t.Fatal("extractor must not run for missing video")
	}
}

func TestThumbnailUnknownEpisodeReturns404(t *testing.T) {
	env := setupThumbnailTest(t)

	response := env.get(t, "/api/catalog/episode/999/thumbnail")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestThumbnailInvalidID(t *testing.T) {
	env := setupThumbnailTest(t)

	response := env.get(t, "/api/catalog/episode/abc/thumbnail")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}
