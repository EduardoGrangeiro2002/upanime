package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/jobs"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

type fakeWorkerClient struct {
	err  error
	jobs []service.UpscaleWorkerJob
}

func (f *fakeWorkerClient) Enqueue(_ context.Context, job service.UpscaleWorkerJob) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.jobs = append(f.jobs, job)
	return fmt.Sprintf("runpod-%d", job.JobID), nil
}

func (f *fakeWorkerClient) Status(_ context.Context, runpodJobID string) (*service.RunPodJobStatus, error) {
	return &service.RunPodJobStatus{ID: runpodJobID, Status: "IN_PROGRESS"}, nil
}

type editionEnv struct {
	h        *handler.EditionHandler
	animes   *store.SQLiteAnimeStore
	episodes *store.SQLiteEpisodeStore
	upscales *store.SQLiteUpscaleStore
	worker   *fakeWorkerClient
	enq      *fakeEnqueuer
}

func setupEditionTest(t *testing.T) *editionEnv {
	t.Helper()

	db := testutil.NewTestDB(t)
	env := &editionEnv{
		animes:   store.NewSQLiteAnimeStore(db),
		episodes: store.NewSQLiteEpisodeStore(db),
		upscales: store.NewSQLiteUpscaleStore(db),
		worker:   &fakeWorkerClient{},
		enq:      &fakeEnqueuer{},
	}
	env.h = handler.NewEditionHandler(
		env.upscales,
		env.animes,
		env.episodes,
		storage.NewLocalStorage(t.TempDir()),
		env.worker,
		env.enq,
	)
	return env
}

func createAnimeWithStorageKey(t *testing.T, animeStore *store.SQLiteAnimeStore, episodeStore *store.SQLiteEpisodeStore) *model.Anime {
	t.Helper()

	anime := &model.Anime{
		Title:     "Upscale Handler Anime",
		URL:       "https://animesonlinecc.to/anime/upscale-handler",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Label: "Season 1", Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/uh-ep1", Type: "episode"},
				{Title: "Ep 2", Number: "2", URL: "https://example.com/uh-ep2", Type: "episode"},
			}},
		},
	}

	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	episodeID := anime.Seasons[0].Episodes[0].ID.Int64()
	if err := episodeStore.UpdateStorageKey(t.Context(), episodeID, "animes/upscale_handler/ep_1.mp4"); err != nil {
		t.Fatalf("set storage key: %v", err)
	}

	return anime
}

func (e *editionEnv) createUpscaleJob(t *testing.T, anime *model.Anime) *model.UpscaleJob {
	t.Helper()
	job := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[0].ID,
		AnimeID:          anime.ID,
		Type:             "upscale",
		SourceStorageKey: "animes/upscale_handler/ep_1.mp4",
		ResultStorageKey: "animes/upscale_handler/ep_1_upscaled.mp4",
	}
	if err := e.upscales.Create(t.Context(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	return job
}

func TestEditionHandler_CreateUpscale_Success(t *testing.T) {
	env := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)

	panRatio := 0.75
	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:      anime.ID,
		EpisodeIDs:   []model.StringID{anime.Seasons[0].Episodes[0].ID},
		TargetHeight: 2160,
		Interpolate:  true,
		PanRatio:     &panRatio,
		Effects:      true,
		SkipUpscale:  true,
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	env.h.Create(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var created []model.UpscaleJob
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 job, got %d", len(created))
	}
	if created[0].Status != "queued" {
		t.Errorf("expected status 'queued', got '%s'", created[0].Status)
	}
	if created[0].TargetHeight != 2160 {
		t.Errorf("expected target height 2160, got %d", created[0].TargetHeight)
	}
	if created[0].ResultStorageKey != "animes/upscale_handler/ep_1_upscaled.mp4" {
		t.Errorf("unexpected result key: %s", created[0].ResultStorageKey)
	}

	if len(env.enq.upscales) != 1 {
		t.Fatalf("expected 1 dispatch enqueued, got %d", len(env.enq.upscales))
	}
	wj := env.enq.upscales[0]
	if wj.SourceStorageKey != "animes/upscale_handler/ep_1.mp4" {
		t.Errorf("unexpected source key: %s", wj.SourceStorageKey)
	}
	if wj.ResultStorageKey != "animes/upscale_handler/ep_1_upscaled.mp4" {
		t.Errorf("unexpected worker result key: %s", wj.ResultStorageKey)
	}
	if wj.TargetHeight != 2160 {
		t.Errorf("unexpected worker target height: %d", wj.TargetHeight)
	}
	if !wj.Interpolate {
		t.Error("expected interpolate to reach the worker job")
	}
	if wj.PanRatio == nil || *wj.PanRatio != 0.75 {
		t.Errorf("expected pan ratio 0.75 to reach the worker job, got %v", wj.PanRatio)
	}
	if !wj.Effects {
		t.Error("expected effects to reach the worker job")
	}
	if !wj.SkipUpscale {
		t.Error("expected skipUpscale to reach the worker job")
	}
	if wj.SourceURL != "" {
		t.Errorf("expected empty source url at enqueue time, got %s", wj.SourceURL)
	}
}

func TestEditionHandler_Create_EnqueueErrorMarksFailed(t *testing.T) {
	env := setupEditionTest(t)
	env.enq.err = errors.New("redis fora")
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)

	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:    anime.ID,
		EpisodeIDs: []model.StringID{anime.Seasons[0].Episodes[0].ID},
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	env.h.Create(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var created []model.UpscaleJob
	json.NewDecoder(response.Body).Decode(&created)
	saved, err := env.upscales.GetByID(t.Context(), created[0].ID.Int64())
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if saved.Status != "failed" {
		t.Fatalf("expected failed on enqueue error, got %s", saved.Status)
	}
}

func TestProcessDispatchTask_Completes(t *testing.T) {
	env := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)
	job := env.createUpscaleJob(t, anime)

	task, err := jobs.NewUpscaleDispatchTask(service.UpscaleWorkerJob{
		JobID:            job.ID.Int64(),
		SourceStorageKey: job.SourceStorageKey,
		ResultStorageKey: job.ResultStorageKey,
		TargetHeight:     1080,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := env.h.ProcessDispatchTask(t.Context(), task); err != nil {
		t.Fatalf("process dispatch: %v", err)
	}

	saved, _ := env.upscales.GetByID(t.Context(), job.ID.Int64())
	if saved.Status != "processing" {
		t.Fatalf("expected processing, got %s (%s)", saved.Status, saved.Error)
	}
	if saved.RunPodJobID == "" {
		t.Fatal("expected runpod job id recorded")
	}
	if len(env.worker.jobs) != 1 {
		t.Fatalf("expected 1 worker enqueue, got %d", len(env.worker.jobs))
	}
	if env.worker.jobs[0].SourceURL == "" {
		t.Fatal("expected source url presigned at dispatch time")
	}
}

func TestProcessDispatchTask_AlreadyDispatchedIsNoop(t *testing.T) {
	env := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)
	job := env.createUpscaleJob(t, anime)
	_ = env.upscales.UpdateRunPodJobID(t.Context(), job.ID.Int64(), "runpod-existente")

	task, _ := jobs.NewUpscaleDispatchTask(service.UpscaleWorkerJob{JobID: job.ID.Int64()})
	if err := env.h.ProcessDispatchTask(t.Context(), task); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(env.worker.jobs) != 0 {
		t.Fatalf("expected no worker enqueue, got %d", len(env.worker.jobs))
	}
}

func TestProcessDispatchTask_DeletedRowIsNoop(t *testing.T) {
	env := setupEditionTest(t)

	task, _ := jobs.NewUpscaleDispatchTask(service.UpscaleWorkerJob{JobID: 9999})
	if err := env.h.ProcessDispatchTask(t.Context(), task); err != nil {
		t.Fatalf("expected nil for missing job, got %v", err)
	}
	if len(env.worker.jobs) != 0 {
		t.Fatalf("expected no worker enqueue, got %d", len(env.worker.jobs))
	}
}

func TestProcessDispatchTask_WorkerFailureMarksFailed(t *testing.T) {
	env := setupEditionTest(t)
	env.worker.err = errors.New("runpod fora")
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)
	job := env.createUpscaleJob(t, anime)

	task, _ := jobs.NewUpscaleDispatchTask(service.UpscaleWorkerJob{
		JobID:            job.ID.Int64(),
		SourceStorageKey: job.SourceStorageKey,
	})
	if err := env.h.ProcessDispatchTask(t.Context(), task); err == nil {
		t.Fatal("expected error to propagate for retry accounting")
	}

	saved, _ := env.upscales.GetByID(t.Context(), job.ID.Int64())
	if saved.Status != "failed" {
		t.Fatalf("expected failed, got %s", saved.Status)
	}
}

func TestEditionHandler_Create_EpisodeNotDownloaded(t *testing.T) {
	env := setupEditionTest(t)

	anime := &model.Anime{
		Title:     "No Download Anime",
		URL:       "https://animesonlinecc.to/anime/no-download",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Label: "Season 1", Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/nd-ep1", Type: "episode"},
			}},
		},
	}
	if err := env.animes.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:    anime.ID,
		EpisodeIDs: []model.StringID{anime.Seasons[0].Episodes[0].ID},
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	env.h.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEditionHandler_Create_InvalidTargetHeight(t *testing.T) {
	env := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)

	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:      anime.ID,
		EpisodeIDs:   []model.StringID{anime.Seasons[0].Episodes[0].ID},
		TargetHeight: 999,
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	env.h.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEditionHandler_List_Empty(t *testing.T) {
	env := setupEditionTest(t)

	request := httptest.NewRequest("GET", "/api/upscale", nil)
	response := httptest.NewRecorder()
	env.h.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var listed []model.UpscaleJob
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(listed))
	}
}

func TestEditionHandler_Delete(t *testing.T) {
	env := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, env.animes, env.episodes)
	job := env.createUpscaleJob(t, anime)

	router := chi.NewRouter()
	router.Delete("/api/upscale/{id}", env.h.Delete)

	request := httptest.NewRequest("DELETE", fmt.Sprintf("/api/upscale/%d", job.ID.Int64()), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}

	if _, err := env.upscales.GetByID(t.Context(), job.ID.Int64()); err == nil {
		t.Fatal("expected error after delete")
	}
	if len(env.enq.upscaleCancelled) != 1 || env.enq.upscaleCancelled[0] != job.ID.Int64() {
		t.Fatalf("expected cancel for %d, got %v", job.ID.Int64(), env.enq.upscaleCancelled)
	}
}
