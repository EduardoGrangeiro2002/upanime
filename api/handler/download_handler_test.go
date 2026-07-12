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

func TestDownloadHandler_CreateAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	dlStore := store.NewSQLiteDownloadStore(db)
	epStore := store.NewSQLiteEpisodeStore(db)
	fs := storage.NewLocalStorage(t.TempDir())

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
	_ = animeStore.Create(t.Context(), anime)

	exec := &fakeExecutor{}
	classifier := service.NewGenreClassifier("", "", "", animeStore)
	h := handler.NewDownloadHandler(dlStore, animeStore, epStore, exec, fs, classifier, ":memory:", 3)

	body, _ := json.Marshal(model.CreateDownloadsRequest{
		AnimeID:    anime.ID,
		AnimeTitle: anime.Title,
		EpisodeIDs: []model.StringID{anime.Seasons[0].Episodes[0].ID},
	})

	req := httptest.NewRequest("POST", "/api/downloads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/api/downloads", nil)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}
}

func TestDownloadHandler_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	dlStore := store.NewSQLiteDownloadStore(db)
	epStore := store.NewSQLiteEpisodeStore(db)
	fs := storage.NewLocalStorage(t.TempDir())

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

	exec := &fakeExecutor{}
	classifier := service.NewGenreClassifier("", "", "", animeStore)
	h := handler.NewDownloadHandler(dlStore, animeStore, epStore, exec, fs, classifier, ":memory:", 3)

	r := chi.NewRouter()
	r.Delete("/api/downloads/{id}", h.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/downloads/%d", downloads[0].ID.Int64()), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
