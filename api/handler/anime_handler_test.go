package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

type fakeExecutor struct {
	result *model.Anime
	err    error
}

func (f *fakeExecutor) Scrape(_ context.Context, _ string) (*model.Anime, error) {
	return f.result, f.err
}

func (f *fakeExecutor) Download(_ context.Context, _ string, _ string, _ int64, _ string) error {
	return nil
}

func TestAnimeHandler_Get_Scrapes(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	scraperStore := store.NewSQLiteScraperStore(db)
	fs := storage.NewLocalStorage(t.TempDir())

	exec := &fakeExecutor{
		result: &model.Anime{
			Title:       "Naruto",
			URL:         "https://animesonlinecc.to/anime/naruto",
			ImageURL:    "https://example.com/naruto.jpg",
			Description: "Ninja anime",
			Seasons: []model.Season{
				{Number: 1, Type: "episode", Episodes: []model.Episode{
					{Title: "Ep 1", Number: "1", URL: "https://animesonlinecc.to/episodio/naruto-1", Type: "episode"},
				}},
			},
		},
	}

	h := handler.NewAnimeHandler(animeStore, scraperStore, exec, fs)

	req := httptest.NewRequest("GET", "/api/anime?url=https://animesonlinecc.to/anime/naruto", nil)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var anime model.Anime
	if err := json.NewDecoder(w.Body).Decode(&anime); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if anime.Title != "Naruto" {
		t.Errorf("expected 'Naruto', got '%s'", anime.Title)
	}
}

func TestAnimeHandler_Get_ReturnsCached(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	scraperStore := store.NewSQLiteScraperStore(db)
	fs := storage.NewLocalStorage(t.TempDir())

	anime := &model.Anime{
		Title:     "Cached Anime",
		URL:       "https://animesonlinecc.to/anime/cached",
		ScraperID: 1,
		Seasons:   []model.Season{{Number: 1, Type: "episode"}},
	}
	_ = animeStore.Create(context.Background(), anime)

	exec := &fakeExecutor{}
	h := handler.NewAnimeHandler(animeStore, scraperStore, exec, fs)

	req := httptest.NewRequest("GET", "/api/anime?url=https://animesonlinecc.to/anime/cached", nil)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result model.Anime
	json.NewDecoder(w.Body).Decode(&result)
	if result.Title != "Cached Anime" {
		t.Errorf("expected 'Cached Anime', got '%s'", result.Title)
	}
}

func TestAnimeHandler_Get_MissingURL(t *testing.T) {
	h := handler.NewAnimeHandler(nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/anime", nil)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
