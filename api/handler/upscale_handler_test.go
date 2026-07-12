package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
	"upanime/api/testutil"
)

type fakeWorkerClient struct {
	err  error
	jobs chan service.UpscaleWorkerJob
}

func (f *fakeWorkerClient) Enqueue(_ context.Context, job service.UpscaleWorkerJob) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	f.jobs <- job
	return fmt.Sprintf("runpod-%d", job.JobID), nil
}

func (f *fakeWorkerClient) Status(_ context.Context, runpodJobID string) (*service.RunPodJobStatus, error) {
	return &service.RunPodJobStatus{ID: runpodJobID, Status: "IN_PROGRESS"}, nil
}

func setupEditionTest(t *testing.T) (
	*handler.EditionHandler,
	*store.SQLiteAnimeStore,
	*store.SQLiteEpisodeStore,
	*store.SQLiteUpscaleStore,
	*fakeWorkerClient,
) {
	t.Helper()

	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	episodeStore := store.NewSQLiteEpisodeStore(db)
	upscaleStore := store.NewSQLiteUpscaleStore(db)
	fileStorage := storage.NewLocalStorage(t.TempDir())
	workerClient := &fakeWorkerClient{jobs: make(chan service.UpscaleWorkerJob, 4)}

	editionHandler := handler.NewEditionHandler(
		upscaleStore,
		animeStore,
		episodeStore,
		fileStorage,
		workerClient,
	)

	return editionHandler, animeStore, episodeStore, upscaleStore, workerClient
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

func TestEditionHandler_CreateUpscale_Success(t *testing.T) {
	editionHandler, animeStore, episodeStore, _, workerClient := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, animeStore, episodeStore)

	panRatio := 0.75
	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:      anime.ID,
		EpisodeIDs:   []model.StringID{anime.Seasons[0].Episodes[0].ID},
		TargetHeight: 2160,
		Interpolate:  true,
		PanRatio:     &panRatio,
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	editionHandler.Create(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var jobs []model.UpscaleJob
	if err := json.NewDecoder(response.Body).Decode(&jobs); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != "queued" {
		t.Errorf("expected status 'queued', got '%s'", jobs[0].Status)
	}
	if jobs[0].TargetHeight != 2160 {
		t.Errorf("expected target height 2160, got %d", jobs[0].TargetHeight)
	}
	if jobs[0].ResultStorageKey != "animes/upscale_handler/ep_1_upscaled.mp4" {
		t.Errorf("unexpected result key: %s", jobs[0].ResultStorageKey)
	}

	select {
	case queuedJob := <-workerClient.jobs:
		if queuedJob.SourceStorageKey != "animes/upscale_handler/ep_1.mp4" {
			t.Errorf("unexpected source key: %s", queuedJob.SourceStorageKey)
		}
		if queuedJob.ResultStorageKey != "animes/upscale_handler/ep_1_upscaled.mp4" {
			t.Errorf("unexpected worker result key: %s", queuedJob.ResultStorageKey)
		}
		if queuedJob.TargetHeight != 2160 {
			t.Errorf("unexpected worker target height: %d", queuedJob.TargetHeight)
		}
		if !queuedJob.Interpolate {
			t.Error("expected interpolate to reach the worker job")
		}
		if queuedJob.PanRatio == nil || *queuedJob.PanRatio != 0.75 {
			t.Errorf("expected pan ratio 0.75 to reach the worker job, got %v", queuedJob.PanRatio)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected job to be queued in worker client")
	}
}

func TestEditionHandler_Create_EpisodeNotDownloaded(t *testing.T) {
	editionHandler, animeStore, _, _, _ := setupEditionTest(t)

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
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:    anime.ID,
		EpisodeIDs: []model.StringID{anime.Seasons[0].Episodes[0].ID},
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	editionHandler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEditionHandler_Create_InvalidTargetHeight(t *testing.T) {
	editionHandler, animeStore, episodeStore, _, _ := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, animeStore, episodeStore)

	body, _ := json.Marshal(model.CreateUpscaleRequest{
		AnimeID:      anime.ID,
		EpisodeIDs:   []model.StringID{anime.Seasons[0].Episodes[0].ID},
		TargetHeight: 999,
	})

	request := httptest.NewRequest("POST", "/api/upscale", bytes.NewReader(body))
	response := httptest.NewRecorder()
	editionHandler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEditionHandler_List_Empty(t *testing.T) {
	editionHandler, _, _, _, _ := setupEditionTest(t)

	request := httptest.NewRequest("GET", "/api/upscale", nil)
	response := httptest.NewRecorder()
	editionHandler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var jobs []model.UpscaleJob
	if err := json.NewDecoder(response.Body).Decode(&jobs); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestEditionHandler_Delete(t *testing.T) {
	editionHandler, animeStore, episodeStore, upscaleStore, _ := setupEditionTest(t)
	anime := createAnimeWithStorageKey(t, animeStore, episodeStore)

	job := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[0].ID,
		AnimeID:          anime.ID,
		Type:             "upscale",
		SourceStorageKey: "animes/upscale_handler/ep_1.mp4",
		ResultStorageKey: "animes/upscale_handler/ep_1_upscaled.mp4",
	}
	if err := upscaleStore.Create(t.Context(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	router := chi.NewRouter()
	router.Delete("/api/upscale/{id}", editionHandler.Delete)

	request := httptest.NewRequest("DELETE", fmt.Sprintf("/api/upscale/%d", job.ID.Int64()), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}

	if _, err := upscaleStore.GetByID(t.Context(), job.ID.Int64()); err == nil {
		t.Fatal("expected error after delete")
	}
}
