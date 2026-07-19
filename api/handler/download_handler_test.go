package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/jobs"
	"upanime/api/model"
	"upanime/api/scraper"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

type fakeEnqueuer struct {
	enqueued         []int64
	cancelled        []int64
	upscales         []service.UpscaleWorkerJob
	upscaleCancelled []int64
	classifyAlls     int
	organized        []int64
	err              error
}

func (f *fakeEnqueuer) EnqueueDownload(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	f.enqueued = append(f.enqueued, id)
	return nil
}

func (f *fakeEnqueuer) CancelDownload(_ context.Context, id int64) error {
	f.cancelled = append(f.cancelled, id)
	return nil
}

func (f *fakeEnqueuer) EnqueueUpscaleDispatch(_ context.Context, wj service.UpscaleWorkerJob) error {
	if f.err != nil {
		return f.err
	}
	f.upscales = append(f.upscales, wj)
	return nil
}

func (f *fakeEnqueuer) CancelUpscale(_ context.Context, id int64) error {
	f.upscaleCancelled = append(f.upscaleCancelled, id)
	return nil
}

func (f *fakeEnqueuer) EnqueueClassifyAll(_ context.Context) error {
	if f.err != nil {
		return f.err
	}
	f.classifyAlls++
	return nil
}

func (f *fakeEnqueuer) EnqueueOrganize(_ context.Context, animeID int64) error {
	if f.err != nil {
		return f.err
	}
	f.organized = append(f.organized, animeID)
	return nil
}

type writingExecutor struct {
	err error
}

func (f *writingExecutor) Scrape(_ context.Context, _ string) (*model.Anime, error) {
	return nil, nil
}

func (f *writingExecutor) Download(_ context.Context, _ string, destPath string, _ int64, _ string) error {
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(destPath, []byte("video"), 0o644)
}

type downloadEnv struct {
	h          *handler.DownloadHandler
	animes     *store.SQLiteAnimeStore
	downloads  *store.SQLiteDownloadStore
	episodes   *store.SQLiteEpisodeStore
	enq        *fakeEnqueuer
	storageDir string
}

func newDownloadEnv(t *testing.T, exec scraper.Executor) *downloadEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	env := &downloadEnv{
		animes:     animeStore,
		downloads:  store.NewSQLiteDownloadStore(db),
		episodes:   store.NewSQLiteEpisodeStore(db),
		enq:        &fakeEnqueuer{},
		storageDir: t.TempDir(),
	}
	env.h = handler.NewDownloadHandler(
		env.downloads,
		animeStore,
		env.episodes,
		store.NewSQLiteScraperStore(db),
		exec,
		storage.NewLocalStorage(env.storageDir),
		service.NewGenreClassifier("", "", "", animeStore),
		":memory:",
		env.enq,
	)
	return env
}

func setupDownloadHandler(t *testing.T) *downloadEnv {
	t.Helper()
	return newDownloadEnv(t, &fakeExecutor{})
}

func (e *downloadEnv) seedAnimeWithEpisode(t *testing.T, title, epURL string) *model.Anime {
	t.Helper()
	anime := &model.Anime{
		Title:     title,
		URL:       "https://animesonlinecc.to/anime/" + title,
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: epURL, Type: "episode"},
			}},
		},
	}
	if err := e.animes.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}
	return anime
}

func (e *downloadEnv) createDownload(t *testing.T, anime *model.Anime) model.Download {
	t.Helper()
	downloads, err := e.downloads.Create(t.Context(), []model.Download{
		{EpisodeID: anime.Seasons[0].Episodes[0].ID, AnimeID: anime.ID},
	})
	if err != nil {
		t.Fatalf("create download: %v", err)
	}
	return downloads[0]
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
	env := setupDownloadHandler(t)

	anime := env.seedAnimeWithEpisode(t, "DL Handler Anime", "https://example.com/ep-handler-1")

	w := postDownloads(t, env.h, model.CreateDownloadsRequest{
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
	if len(env.enq.enqueued) != 2 {
		t.Fatalf("expected 2 tasks enqueued, got %d", len(env.enq.enqueued))
	}

	saved, _ := env.animes.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Seasons) != 1 {
		t.Fatalf("expected 1 season, got %d", len(saved.Seasons))
	}
	if len(saved.Seasons[0].Episodes) != 2 {
		t.Errorf("expected existing episode reused and new one added, got %d episodes", len(saved.Seasons[0].Episodes))
	}
}

func TestDownloadHandler_Create_EnqueueErrorMarksFailed(t *testing.T) {
	env := setupDownloadHandler(t)
	env.enq.err = errors.New("redis fora")

	anime := env.seedAnimeWithEpisode(t, "Enqueue Fail Anime", "https://example.com/ep-enq-1")

	w := postDownloads(t, env.h, model.CreateDownloadsRequest{
		AnimeID:   anime.ID,
		SourceURL: anime.URL,
		Episodes: []model.DownloadEpisodeInput{
			{Title: "Ep 1", Number: "1", URL: "https://example.com/ep-enq-1", SeasonNumber: 1},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var downloads []model.Download
	json.NewDecoder(w.Body).Decode(&downloads)
	saved, err := env.downloads.GetByID(t.Context(), downloads[0].ID.Int64())
	if err != nil {
		t.Fatalf("get download: %v", err)
	}
	if saved.Status != "failed" {
		t.Fatalf("expected failed on enqueue error, got %s", saved.Status)
	}
}

func TestDownloadHandler_Create_NewAnimeByTitle(t *testing.T) {
	env := setupDownloadHandler(t)

	w := postDownloads(t, env.h, model.CreateDownloadsRequest{
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

	saved, err := env.animes.FindByTitle(t.Context(), "Slayers Completo")
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
	env := setupDownloadHandler(t)

	anime := env.seedAnimeWithEpisode(t, "Slayers", "https://example.com/slayers-s1-1")

	w := postDownloads(t, env.h, model.CreateDownloadsRequest{
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

	saved, _ := env.animes.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Seasons) != 2 {
		t.Fatalf("expected 2 seasons, got %d", len(saved.Seasons))
	}
	if len(saved.Seasons[1].Episodes) != 1 {
		t.Errorf("expected 1 episode in season 2, got %d", len(saved.Seasons[1].Episodes))
	}
}

func TestDownloadHandler_Create_MissingTarget(t *testing.T) {
	env := setupDownloadHandler(t)

	w := postDownloads(t, env.h, model.CreateDownloadsRequest{
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
	env := setupDownloadHandler(t)

	w := postDownloads(t, env.h, model.CreateDownloadsRequest{AnimeTitle: "X", SourceURL: "https://animesonlinecc.to/anime/x"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDownloadHandler_Delete(t *testing.T) {
	env := setupDownloadHandler(t)

	anime := env.seedAnimeWithEpisode(t, "Del Handler Anime", "https://example.com/ep-del-handler-1")
	d := env.createDownload(t, anime)

	r := chi.NewRouter()
	r.Delete("/api/downloads/{id}", env.h.Delete)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/downloads/%d", d.ID.Int64()), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if len(env.enq.cancelled) != 1 || env.enq.cancelled[0] != d.ID.Int64() {
		t.Fatalf("expected task cancelled for %d, got %v", d.ID.Int64(), env.enq.cancelled)
	}
}

func TestProcessDownloadTask_Completes(t *testing.T) {
	env := newDownloadEnv(t, &writingExecutor{})

	anime := env.seedAnimeWithEpisode(t, "Task Anime", "https://example.com/ep-task-1")
	d := env.createDownload(t, anime)

	if err := env.h.ProcessDownloadTask(t.Context(), jobs.NewDownloadTask(d.ID.Int64())); err != nil {
		t.Fatalf("process task: %v", err)
	}

	saved, _ := env.downloads.GetByID(t.Context(), d.ID.Int64())
	if saved.Status != "completed" {
		t.Fatalf("expected completed, got %s (%s)", saved.Status, saved.Error)
	}

	ep, err := env.episodes.GetByID(t.Context(), anime.Seasons[0].Episodes[0].ID.Int64())
	if err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if ep.StorageKey == "" {
		t.Fatal("expected episode storage key set")
	}
	if _, err := os.Stat(filepath.Join(env.storageDir, ep.StorageKey)); err != nil {
		t.Fatalf("expected file saved in storage: %v", err)
	}
}

func TestProcessDownloadTask_ExecutorFailureMarksFailed(t *testing.T) {
	env := newDownloadEnv(t, &writingExecutor{err: errors.New("fonte fora do ar")})

	anime := env.seedAnimeWithEpisode(t, "Task Fail Anime", "https://example.com/ep-taskfail-1")
	d := env.createDownload(t, anime)

	if err := env.h.ProcessDownloadTask(t.Context(), jobs.NewDownloadTask(d.ID.Int64())); err == nil {
		t.Fatal("expected error to propagate for retry accounting")
	}

	saved, _ := env.downloads.GetByID(t.Context(), d.ID.Int64())
	if saved.Status != "failed" {
		t.Fatalf("expected failed, got %s", saved.Status)
	}
	if saved.Error == "" {
		t.Fatal("expected error message recorded")
	}
}

func TestProcessDownloadTask_DeletedRowIsNoop(t *testing.T) {
	env := newDownloadEnv(t, &writingExecutor{})

	if err := env.h.ProcessDownloadTask(t.Context(), jobs.NewDownloadTask(9999)); err != nil {
		t.Fatalf("expected nil for missing download, got %v", err)
	}
}

func TestProcessDownloadTask_CompletedRowIsNoop(t *testing.T) {
	exec := &writingExecutor{err: errors.New("não deveria ser chamado")}
	env := newDownloadEnv(t, exec)

	anime := env.seedAnimeWithEpisode(t, "Task Done Anime", "https://example.com/ep-taskdone-1")
	d := env.createDownload(t, anime)
	_ = env.downloads.UpdateStatus(t.Context(), d.ID.Int64(), "completed", "")

	if err := env.h.ProcessDownloadTask(t.Context(), jobs.NewDownloadTask(d.ID.Int64())); err != nil {
		t.Fatalf("expected nil for completed download, got %v", err)
	}
}
