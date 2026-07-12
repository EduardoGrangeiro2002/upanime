package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

func setupUploadTest(t *testing.T) (*handler.UploadHandler, *store.SQLiteAnimeStore, string) {
	t.Helper()
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	epStore := store.NewSQLiteEpisodeStore(db)
	scraperStore := store.NewSQLiteScraperStore(db)
	dir := t.TempDir()
	fs := storage.NewLocalStorage(dir)
	classifier := service.NewGenreClassifier("", "", "", animeStore)
	h := handler.NewUploadHandler(animeStore, epStore, scraperStore, fs, classifier)
	return h, animeStore, dir
}

func uploadRequest(t *testing.T, fields map[string]string, filename, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range fields {
		writer.WriteField(k, v)
	}
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		io.WriteString(part, content)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/api/catalog/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadHandler_CreatesAnimeSeasonAndEpisode(t *testing.T) {
	h, animeStore, dir := setupUploadTest(t)

	req := uploadRequest(t, map[string]string{
		"animeTitle":    "Upload Anime",
		"seasonNumber":  "1",
		"episodeNumber": "1",
	}, "ep1.mp4", "video-bytes")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		AnimeID  model.StringID `json:"animeId"`
		Episode  model.Episode  `json:"episode"`
		Replaced bool           `json:"replaced"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Replaced {
		t.Fatal("expected replaced=false on first upload")
	}
	if resp.Episode.StorageKey == "" {
		t.Fatal("expected episode storage key to be set")
	}

	anime, err := animeStore.GetByID(t.Context(), resp.AnimeID.Int64())
	if err != nil {
		t.Fatalf("anime not persisted: %v", err)
	}
	if len(anime.Seasons) != 1 || len(anime.Seasons[0].Episodes) != 1 {
		t.Fatalf("expected 1 season with 1 episode, got %+v", anime.Seasons)
	}
	if anime.Seasons[0].Episodes[0].Title != "Episódio 1" {
		t.Fatalf("expected default episode title, got %q", anime.Seasons[0].Episodes[0].Title)
	}

	filePath := filepath.Join(dir, resp.Episode.StorageKey)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("uploaded file not saved: %v", err)
	}
	if string(data) != "video-bytes" {
		t.Fatalf("unexpected file content %q", data)
	}
}

func TestUploadHandler_ReusesAnimeAndReplacesEpisode(t *testing.T) {
	h, animeStore, dir := setupUploadTest(t)

	first := uploadRequest(t, map[string]string{
		"animeTitle":    "Repeat Anime",
		"episodeNumber": "1",
	}, "ep1.mp4", "original")
	w1 := httptest.NewRecorder()
	h.Create(w1, first)
	if w1.Code != http.StatusOK {
		t.Fatalf("first upload failed: %d %s", w1.Code, w1.Body.String())
	}

	second := uploadRequest(t, map[string]string{
		"animeTitle":    "Repeat Anime",
		"episodeNumber": "2",
	}, "ep2.mp4", "segundo")
	w2 := httptest.NewRecorder()
	h.Create(w2, second)
	if w2.Code != http.StatusOK {
		t.Fatalf("second upload failed: %d %s", w2.Code, w2.Body.String())
	}

	replay := uploadRequest(t, map[string]string{
		"animeTitle":    "Repeat Anime",
		"episodeNumber": "1",
	}, "ep1-v2.mp4", "replaced-content")
	w3 := httptest.NewRecorder()
	h.Create(w3, replay)
	if w3.Code != http.StatusOK {
		t.Fatalf("replay upload failed: %d %s", w3.Code, w3.Body.String())
	}

	var resp struct {
		AnimeID  model.StringID `json:"animeId"`
		Episode  model.Episode  `json:"episode"`
		Replaced bool           `json:"replaced"`
	}
	json.NewDecoder(w3.Body).Decode(&resp)
	if !resp.Replaced {
		t.Fatal("expected replaced=true when re-uploading same episode number")
	}

	anime, err := animeStore.GetByID(t.Context(), resp.AnimeID.Int64())
	if err != nil {
		t.Fatalf("load anime: %v", err)
	}
	if len(anime.Seasons) != 1 {
		t.Fatalf("expected single season, got %d", len(anime.Seasons))
	}
	if len(anime.Seasons[0].Episodes) != 2 {
		t.Fatalf("expected 2 episodes after replay, got %d", len(anime.Seasons[0].Episodes))
	}

	data, err := os.ReadFile(filepath.Join(dir, resp.Episode.StorageKey))
	if err != nil {
		t.Fatalf("replaced file not saved: %v", err)
	}
	if string(data) != "replaced-content" {
		t.Fatalf("expected replaced content, got %q", data)
	}
}

func TestUploadHandler_Validation(t *testing.T) {
	h, _, _ := setupUploadTest(t)

	cases := []struct {
		name     string
		fields   map[string]string
		filename string
	}{
		{"missing title", map[string]string{"episodeNumber": "1"}, "ep.mp4"},
		{"missing episode number", map[string]string{"animeTitle": "X"}, "ep.mp4"},
		{"missing file", map[string]string{"animeTitle": "X", "episodeNumber": "1"}, ""},
		{"bad extension", map[string]string{"animeTitle": "X", "episodeNumber": "1"}, "ep.txt"},
		{"bad season", map[string]string{"animeTitle": "X", "episodeNumber": "1", "seasonNumber": "abc"}, "ep.mp4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := uploadRequest(t, tc.fields, tc.filename, "data")
			w := httptest.NewRecorder()
			h.Create(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestUploadHandler_UploadedAnimeAppearsInCatalog(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	epStore := store.NewSQLiteEpisodeStore(db)
	scraperStore := store.NewSQLiteScraperStore(db)
	dir := t.TempDir()
	fs := storage.NewLocalStorage(dir)
	classifier := service.NewGenreClassifier("", "", "", animeStore)
	uploadH := handler.NewUploadHandler(animeStore, epStore, scraperStore, fs, classifier)
	catalogH := handler.NewCatalogHandler(animeStore, epStore, fs)

	req := uploadRequest(t, map[string]string{
		"animeTitle":    "Catalog Upload",
		"episodeNumber": "3",
	}, "ep3.mp4", "abc")
	w := httptest.NewRecorder()
	uploadH.Create(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed: %d %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/api/catalog", nil)
	listW := httptest.NewRecorder()
	catalogH.List(listW, listReq)

	var animes []model.Anime
	json.NewDecoder(listW.Body).Decode(&animes)
	if len(animes) != 1 {
		t.Fatalf("expected uploaded anime in catalog, got %d animes", len(animes))
	}
	if animes[0].Title != "Catalog Upload" {
		t.Fatalf("unexpected catalog anime %q", animes[0].Title)
	}
}
