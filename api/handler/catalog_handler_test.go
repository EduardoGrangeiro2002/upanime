package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

func setupCatalogTest(t *testing.T) (*handler.CatalogHandler, *store.SQLiteAnimeStore, *store.SQLiteEpisodeStore) {
	t.Helper()
	h, animeStore, epStore, _ := setupCatalogTestWithDir(t)
	return h, animeStore, epStore
}

func setupCatalogTestWithDir(t *testing.T) (*handler.CatalogHandler, *store.SQLiteAnimeStore, *store.SQLiteEpisodeStore, string) {
	t.Helper()
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	epStore := store.NewSQLiteEpisodeStore(db)
	dir := t.TempDir()
	fs := storage.NewLocalStorage(dir)
	h := handler.NewCatalogHandler(animeStore, epStore, fs)
	return h, animeStore, epStore, dir
}

func createAnimeWithEpisodes(t *testing.T, animeStore *store.SQLiteAnimeStore) *model.Anime {
	t.Helper()
	anime := &model.Anime{
		Title:     "Catalog Test Anime",
		URL:       "https://animesonlinecc.to/anime/catalog-test",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Label: "Season 1", Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/ep1", Type: "episode"},
				{Title: "Ep 2", Number: "2", URL: "https://example.com/ep2", Type: "episode"},
			}},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}
	return anime
}

func TestCatalogHandler_List_Empty(t *testing.T) {
	h, _, _ := setupCatalogTest(t)

	req := httptest.NewRequest("GET", "/api/catalog", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result []model.Anime
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 0 {
		t.Fatalf("expected empty catalog, got %d animes", len(result))
	}
}

func TestCatalogHandler_List_OnlyWithDownloads(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	req := httptest.NewRequest("GET", "/api/catalog", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	var result []model.Anime
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 0 {
		t.Fatalf("expected 0 animes (no downloads), got %d", len(result))
	}

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")

	w2 := httptest.NewRecorder()
	h.List(w2, httptest.NewRequest("GET", "/api/catalog", nil))

	var result2 []model.Anime
	json.NewDecoder(w2.Body).Decode(&result2)
	if len(result2) != 1 {
		t.Fatalf("expected 1 anime with downloads, got %d", len(result2))
	}
}

func TestCatalogHandler_DeleteAnime(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")

	r := chi.NewRouter()
	r.Delete("/api/catalog/anime/{id}", h.DeleteAnime)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/catalog/anime/%d", anime.ID.Int64()), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	_, err := animeStore.GetByID(t.Context(), anime.ID.Int64())
	if err == nil {
		t.Fatal("expected anime to be deleted")
	}
}

func TestCatalogHandler_DeleteEpisode(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")
	_ = epStore.UpdateUpscaledStorageKey(t.Context(), epID, "animes/test/ep1_upscaled.mp4")

	r := chi.NewRouter()
	r.Delete("/api/catalog/episode/{id}", h.DeleteEpisode)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/catalog/episode/%d", epID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	ep, err := epStore.GetByID(t.Context(), epID)
	if err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if ep.StorageKey != "" {
		t.Fatalf("expected empty storage key, got '%s'", ep.StorageKey)
	}
	if ep.UpscaledStorageKey != "" {
		t.Fatalf("expected empty upscaled storage key, got '%s'", ep.UpscaledStorageKey)
	}
}

func TestCatalogHandler_DeleteAnime_NotFound(t *testing.T) {
	h, _, _ := setupCatalogTest(t)

	r := chi.NewRouter()
	r.Delete("/api/catalog/anime/{id}", h.DeleteAnime)

	req := httptest.NewRequest("DELETE", "/api/catalog/anime/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCatalogHandler_DeleteEpisode_NotFound(t *testing.T) {
	h, _, _ := setupCatalogTest(t)

	r := chi.NewRouter()
	r.Delete("/api/catalog/episode/{id}", h.DeleteEpisode)

	req := httptest.NewRequest("DELETE", "/api/catalog/episode/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCatalogHandler_StreamURL(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream", h.StreamURL)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/catalog/episode/%d/stream", epID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["url"] == "" {
		t.Fatal("expected url in response body")
	}
}

func TestCatalogHandler_StreamURL_UpscaledVariant(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")
	_ = epStore.UpdateUpscaledStorageKey(t.Context(), epID, "animes/test/ep1_upscaled.mp4")

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream", h.StreamURL)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/catalog/episode/%d/stream?variant=upscaled", epID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["url"] == "" {
		t.Fatal("expected url in response body")
	}
}

func TestCatalogHandler_StreamURL_NoStorageKey(t *testing.T) {
	h, animeStore, _ := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream", h.StreamURL)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/catalog/episode/%d/stream", epID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCatalogHandler_DeleteUpscaledEpisode(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")
	_ = epStore.UpdateUpscaledStorageKey(t.Context(), epID, "animes/test/ep1_upscaled.mp4")

	r := chi.NewRouter()
	r.Delete("/api/catalog/episode/{id}/upscaled", h.DeleteUpscaledEpisode)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/catalog/episode/%d/upscaled", epID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	ep, err := epStore.GetByID(t.Context(), epID)
	if err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if ep.StorageKey == "" {
		t.Fatal("expected original storage key to remain")
	}
	if ep.UpscaledStorageKey != "" {
		t.Fatalf("expected empty upscaled storage key, got '%s'", ep.UpscaledStorageKey)
	}
}

func TestCatalogHandler_StreamURL_NotFound(t *testing.T) {
	h, _, _ := setupCatalogTest(t)

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream", h.StreamURL)

	req := httptest.NewRequest("GET", "/api/catalog/episode/999/stream", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCatalogHandler_UploadCover(t *testing.T) {
	h, animeStore, epStore := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	_ = epStore.UpdateStorageKey(t.Context(), epID, "animes/test/ep1.mp4")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("cover", "cover.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte("fake image data"))
	writer.Close()

	r := chi.NewRouter()
	r.Post("/api/catalog/anime/{id}/cover", h.UploadCover)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/catalog/anime/%d/cover", anime.ID.Int64()), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["coverUrl"] == "" {
		t.Fatal("expected coverUrl in response")
	}

	updated, err := animeStore.GetByID(t.Context(), anime.ID.Int64())
	if err != nil {
		t.Fatalf("get anime: %v", err)
	}
	if updated.CoverPath == "" {
		t.Fatal("expected cover path to be set")
	}
}

func TestCatalogHandler_UploadCover_NotFound(t *testing.T) {
	h, _, _ := setupCatalogTest(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("cover", "cover.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte("fake image data"))
	writer.Close()

	r := chi.NewRouter()
	r.Post("/api/catalog/anime/{id}/cover", h.UploadCover)

	req := httptest.NewRequest("POST", "/api/catalog/anime/999/cover", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCatalogHandler_UploadCover_NoFile(t *testing.T) {
	h, animeStore, _ := setupCatalogTest(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	r := chi.NewRouter()
	r.Post("/api/catalog/anime/{id}/cover", h.UploadCover)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/catalog/anime/%d/cover", anime.ID.Int64()), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func writeTestFile(t *testing.T, dir, key string, data []byte) {
	t.Helper()
	fullPath := filepath.Join(dir, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestCatalogHandler_StreamFile(t *testing.T) {
	h, animeStore, epStore, dir := setupCatalogTestWithDir(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	storageKey := "animes/test/ep1.mp4"
	_ = epStore.UpdateStorageKey(t.Context(), epID, storageKey)

	content := bytes.Repeat([]byte("A"), 1024)
	writeTestFile(t, dir, storageKey, content)

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream/file", h.StreamFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/catalog/episode/%d/stream/file", epID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", w.Body.Len())
	}
}

func TestCatalogHandler_StreamFile_RangeRequest(t *testing.T) {
	h, animeStore, epStore, dir := setupCatalogTestWithDir(t)
	anime := createAnimeWithEpisodes(t, animeStore)

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	storageKey := "animes/test/ep1.mp4"
	_ = epStore.UpdateStorageKey(t.Context(), epID, storageKey)

	content := bytes.Repeat([]byte("B"), 2048)
	writeTestFile(t, dir, storageKey, content)

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream/file", h.StreamFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/catalog/episode/%d/stream/file", epID), nil)
	req.Header.Set("Range", "bytes=0-511")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 512 {
		t.Fatalf("expected 512 bytes, got %d", w.Body.Len())
	}
	if w.Header().Get("Content-Range") == "" {
		t.Fatal("expected Content-Range header")
	}
}

func TestCatalogHandler_StreamFile_NotFound(t *testing.T) {
	h, _, _ := setupCatalogTest(t)

	r := chi.NewRouter()
	r.Get("/api/catalog/episode/{id}/stream/file", h.StreamFile)

	req := httptest.NewRequest("GET", "/api/catalog/episode/999/stream/file", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
