package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/service"
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

func TestAnimeHandler_Get_ScrapesWithoutPersisting(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	scraperStore := store.NewSQLiteScraperStore(db)

	exec := &fakeExecutor{
		result: &model.Anime{
			Title:       "Naruto",
			URL:         "https://animesonlinecc.to/anime/naruto",
			ImageURL:    "https://example.com/naruto.jpg",
			Description: "Ninja anime",
			Seasons: []model.Season{
				{Number: 1, Type: "episode", Episodes: []model.Episode{
					{Title: "Ep 1", Number: "1", URL: "https://animesonlinecc.to/episodio/naruto-1", Type: "episode", SeasonNumber: 1},
				}},
			},
		},
	}

	h := handler.NewAnimeHandler(scraperStore, exec, service.NewEpisodeOrganizer("", "", ""))

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

	saved, err := animeStore.List(t.Context())
	if err != nil {
		t.Fatalf("list animes: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("expected preview not to persist, found %d animes", len(saved))
	}
}

func TestAnimeHandler_Get_UnknownDomain(t *testing.T) {
	db := testutil.NewTestDB(t)
	scraperStore := store.NewSQLiteScraperStore(db)

	h := handler.NewAnimeHandler(scraperStore, &fakeExecutor{}, service.NewEpisodeOrganizer("", "", ""))

	req := httptest.NewRequest("GET", "/api/anime?url=https://unknown-site.com/anime/x", nil)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAnimeHandler_Get_MissingURL(t *testing.T) {
	h := handler.NewAnimeHandler(nil, nil, service.NewEpisodeOrganizer("", "", ""))

	req := httptest.NewRequest("GET", "/api/anime", nil)
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
