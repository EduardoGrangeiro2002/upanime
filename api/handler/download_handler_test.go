package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

func setupDownloadHandler(t *testing.T) (*handler.DownloadHandler, *store.SQLiteAnimeStore, *store.SQLiteDownloadStore) {
	t.Helper()
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	dlStore := store.NewSQLiteDownloadStore(db)
	epStore := store.NewSQLiteEpisodeStore(db)
	scraperStore := store.NewSQLiteScraperStore(db)
	fs := storage.NewLocalStorage(t.TempDir())
	classifier := service.NewGenreClassifier("", "", "", animeStore)
	h := handler.NewDownloadHandler(dlStore, animeStore, epStore, scraperStore, &fakeExecutor{}, fs, classifier, ":memory:", 3)
	return h, animeStore, dlStore
}

func postDownloads(t *testing.T, h *handler.DownloadHandler, req model.CreateDownloadsRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/api/downloads", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, r)
	return w
}

func TestDownloadHandler_Create_ExistingAnime(t *testing.T) {
	h, animeStore, _ := setupDownloadHandler(t)

	anime := &model.Anime{
		Title:     "DL Handler Anime",
		URL:       "https://animesonlinecc.to/anime/dlhandler",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/ep-handler-1", Type: "episode"},
			}},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	w := postDownloads(t, h, model.CreateDownloadsRequest{
		AnimeID:       anime.ID,
		AnimeImageURL: "https://example.com/img.jpg",
		SourceURL:     anime.URL,
		Episodes: []model.DownloadEpisodeInput{
			{Title: "Ep 1", Number: "1", URL: "https://example.com/ep-handler-1", SeasonNumber: 1},
			{Title: "Ep 2", Number: "2", URL: "https://example.com/ep-handler-2", SeasonNumber: 1},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var downloads []model.Download
	if err := json.NewDecoder(w.Body).Decode(&downloads); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(downloads) != 2 {
		t.Fatalf("expected 2 downloads, got %d", len(downloads))
	}
	if downloads[0].AnimeTitle != "DL Handler Anime" {
		t.Errorf("expected anime title, got '%s'", downloads[0].AnimeTitle)
	}

	saved, _ := animeStore.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Seasons) != 1 {
		t.Fatalf("expected 1 season, got %d", len(saved.Seasons))
	}
	if len(saved.Seasons[0].Episodes) != 2 {
		t.Errorf("expected existing episode reused and new one added, got %d episodes", len(saved.Seasons[0].Episodes))
	}
}

func TestDownloadHandler_Create_NewAnimeByTitle(t *testing.T) {
	h, animeStore, _ := setupDownloadHandler(t)

	w := postDownloads(t, h, model.CreateDownloadsRequest{
		AnimeTitle:    "Slayers Completo",
		AnimeImageURL: "https://example.com/slayers.jpg",
		Description:   "Lina Inverse",
		SourceURL:     "https://animesonlinecc.to/anime/slayers",
		Episodes: []model.DownloadEpisodeInput{
			{Title: "Ep 1", Number: "1", URL: "https://example.com/slayers-1", SeasonNumber: 1},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	saved, err := animeStore.FindByTitle(t.Context(), "Slayers Completo")
	if err != nil {
		t.Fatalf("expected anime created: %v", err)
	}
	if saved.Description != "Lina Inverse" {
		t.Errorf("expected description saved, got '%s'", saved.Description)
	}
	if len(saved.Seasons) != 1 || len(saved.Seasons[0].Episodes) != 1 {
		t.Fatalf("expected 1 season with 1 episode, got %+v", saved.Seasons)
	}
}

func TestDownloadHandler_Create_SeasonOverride(t *testing.T) {
	h, animeStore, _ := setupDownloadHandler(t)

	anime := &model.Anime{
		Title:     "Slayers",
		URL:       "https://animesonlinecc.to/anime/slayers",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/slayers-s1-1", Type: "episode"},
			}},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	w := postDownloads(t, h, model.CreateDownloadsRequest{
		AnimeID:      anime.ID,
		SourceURL:    "https://animesonlinecc.to/anime/slayers-next",
		SeasonNumber: 2,
		Episodes: []model.DownloadEpisodeInput{
			{Title: "Ep 1", Number: "1", URL: "https://example.com/slayers-next-1", SeasonNumber: 1},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var downloads []model.Download
	json.NewDecoder(w.Body).Decode(&downloads)
	if downloads[0].SeasonNumber != 2 {
		t.Errorf("expected episode allocated to season 2, got %d", downloads[0].SeasonNumber)
	}

	saved, _ := animeStore.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Seasons) != 2 {
		t.Fatalf("expected 2 seasons, got %d", len(saved.Seasons))
	}
	if len(saved.Seasons[1].Episodes) != 1 {
		t.Errorf("expected 1 episode in season 2, got %d", len(saved.Seasons[1].Episodes))
	}
}

func TestDownloadHandler_Create_MissingTarget(t *testing.T) {
	h, _, _ := setupDownloadHandler(t)

	w := postDownloads(t, h, model.CreateDownloadsRequest{
		SourceURL: "https://animesonlinecc.to/anime/x",
		Episodes: []model.DownloadEpisodeInput{
			{Title: "Ep 1", Number: "1", URL: "https://example.com/x-1", SeasonNumber: 1},
		},
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDownloadHandler_Create_NoEpisodes(t *testing.T) {
	h, _, _ := setupDownloadHandler(t)

	w := postDownloads(t, h, model.CreateDownloadsRequest{AnimeTitle: "X", SourceURL: "https://animesonlinecc.to/anime/x"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDownloadHandler_Delete(t *testing.T) {
	h, animeStore, dlStore := setupDownloadHandler(t)

	anime := &model.Anime{
		Title:     "Del Handler Anime",
		URL:       "https://animesonlinecc.to/anime/delhandler",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/ep-del-handler-1", Type: "episode"},
			}},
		},
	}
	_ = animeStore.Create(t.Context(), anime)

	downloads, _ := dlStore.Create(t.Context(), []model.Download{
		{EpisodeID: anime.Seasons[0].Episodes[0].ID, AnimeID: anime.ID},
	})

	r := chi.NewRouter()
	r.Delete("/api/downloads/{id}", h.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/downloads/%d", downloads[0].ID.Int64()), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
