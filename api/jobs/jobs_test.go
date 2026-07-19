package jobs

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func TestDownloadTaskPayload(t *testing.T) {
	task := NewDownloadTask(42)
	if task.Type() != TypeDownload {
		t.Fatalf("expected %s, got %s", TypeDownload, task.Type())
	}
	var p DownloadPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		t.Fatal(err)
	}
	if p.DownloadID != 42 {
		t.Fatalf("expected 42, got %d", p.DownloadID)
	}
}

func newTestEnqueuer(t *testing.T) (*AsynqEnqueuer, bool) {
	t.Helper()
	addr := os.Getenv("UPANIME_TEST_REDIS_ADDR")
	if addr == "" {
		addr = miniredis.RunT(t).Addr()
	}
	e := NewAsynqEnqueuer(addr)
	t.Cleanup(func() { e.Close() })
	return e, os.Getenv("UPANIME_TEST_REDIS_ADDR") == ""
}

func enqueueOrSkip(t *testing.T, e *AsynqEnqueuer, lenient bool, downloadID int64) {
	t.Helper()
	err := e.EnqueueDownload(t.Context(), downloadID)
	if err == nil {
		return
	}
	if lenient {
		t.Skipf("miniredis incompatível com asynq: %v", err)
	}
	t.Fatal(err)
}

func TestEnqueueAndCancelDownload(t *testing.T) {
	e, lenient := newTestEnqueuer(t)

	enqueueOrSkip(t, e, lenient, 7)

	info, err := e.Inspector().GetTaskInfo(QueueDownloads, DownloadTaskID(7))
	if err != nil {
		t.Fatal(err)
	}
	if info.State != asynq.TaskStatePending {
		t.Fatalf("expected pending, got %s", info.State)
	}

	if err := e.CancelDownload(t.Context(), 7); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Inspector().GetTaskInfo(QueueDownloads, DownloadTaskID(7)); !errors.Is(err, asynq.ErrTaskNotFound) {
		t.Fatalf("expected task deleted, got %v", err)
	}
}

func TestCancelMissingTaskIsNil(t *testing.T) {
	e, _ := newTestEnqueuer(t)

	if err := e.CancelDownload(t.Context(), 12345); err != nil {
		t.Fatalf("expected nil cancelling missing task, got %v", err)
	}
}

func TestReconcileMarksOrphans(t *testing.T) {
	e, lenient := newTestEnqueuer(t)
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	downloads := store.NewSQLiteDownloadStore(db)
	upscales := store.NewSQLiteUpscaleStore(db)

	anime := &model.Anime{
		Title:     "Reconcile Anime",
		URL:       "https://animesonlinecc.to/anime/reconcile",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/rec-1", Type: "episode"},
				{Title: "Ep 2", Number: "2", URL: "https://example.com/rec-2", Type: "episode"},
			}},
		},
	}
	if err := animes.Create(t.Context(), anime); err != nil {
		t.Fatal(err)
	}

	rows, err := downloads.Create(t.Context(), []model.Download{
		{EpisodeID: anime.Seasons[0].Episodes[0].ID, AnimeID: anime.ID},
		{EpisodeID: anime.Seasons[0].Episodes[1].ID, AnimeID: anime.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	enqueueOrSkip(t, e, lenient, rows[1].ID.Int64())

	orphanUpscale := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[0].ID,
		AnimeID:          anime.ID,
		SourceStorageKey: "animes/reconcile/ep_1.mp4",
		ResultStorageKey: "animes/reconcile/ep_1_upscaled.mp4",
	}
	if err := upscales.Create(t.Context(), orphanUpscale); err != nil {
		t.Fatal(err)
	}

	dispatchedUpscale := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[1].ID,
		AnimeID:          anime.ID,
		SourceStorageKey: "animes/reconcile/ep_2.mp4",
		ResultStorageKey: "animes/reconcile/ep_2_upscaled.mp4",
	}
	if err := upscales.Create(t.Context(), dispatchedUpscale); err != nil {
		t.Fatal(err)
	}
	_ = upscales.UpdateRunPodJobID(t.Context(), dispatchedUpscale.ID.Int64(), "runpod-vivo")

	if err := Reconcile(t.Context(), e.Inspector(), downloads, upscales); err != nil {
		t.Fatal(err)
	}

	orphan, _ := downloads.GetByID(t.Context(), rows[0].ID.Int64())
	if orphan.Status != "failed" || orphan.Error != orphanError {
		t.Fatalf("expected orphan failed, got %s (%s)", orphan.Status, orphan.Error)
	}

	tracked, _ := downloads.GetByID(t.Context(), rows[1].ID.Int64())
	if tracked.Status != "queued" {
		t.Fatalf("expected tracked download untouched, got %s", tracked.Status)
	}

	deadUpscale, _ := upscales.GetByID(t.Context(), orphanUpscale.ID.Int64())
	if deadUpscale.Status != "failed" || deadUpscale.Error != orphanError {
		t.Fatalf("expected upscale orphan failed, got %s (%s)", deadUpscale.Status, deadUpscale.Error)
	}

	aliveUpscale, _ := upscales.GetByID(t.Context(), dispatchedUpscale.ID.Int64())
	if aliveUpscale.Status != "queued" {
		t.Fatalf("expected dispatched upscale untouched, got %s", aliveUpscale.Status)
	}
}
