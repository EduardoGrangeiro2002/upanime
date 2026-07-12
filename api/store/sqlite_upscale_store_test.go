package store_test

import (
	"context"
	"testing"

	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func seedAnimeWithStorageKey(t *testing.T, animeStore *store.SQLiteAnimeStore, epStore *store.SQLiteEpisodeStore) *model.Anime {
	t.Helper()
	ctx := context.Background()

	anime := &model.Anime{
		Title:     "Upscale Anime",
		URL:       "https://animesonlinecc.to/anime/upscale",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/up-ep1", Type: "episode"},
				{Title: "Ep 2", Number: "2", URL: "https://example.com/up-ep2", Type: "episode"},
			}},
		},
	}
	if err := animeStore.Create(ctx, anime); err != nil {
		t.Fatalf("seed anime: %v", err)
	}

	epID := anime.Seasons[0].Episodes[0].ID.Int64()
	if err := epStore.UpdateStorageKey(ctx, epID, "animes/upscale_anime/ep_1.mp4"); err != nil {
		t.Fatalf("set storage key: %v", err)
	}

	return anime
}

func TestSQLiteUpscaleStore_CreateAndListActive(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	epStore := store.NewSQLiteEpisodeStore(database)
	upscaleStore := store.NewSQLiteUpscaleStore(database)
	ctx := context.Background()

	anime := seedAnimeWithStorageKey(t, animeStore, epStore)
	epID := anime.Seasons[0].Episodes[0].ID

	job := &model.UpscaleJob{
		EpisodeID:        epID,
		AnimeID:          anime.ID,
		TargetHeight:     1440,
		SourceStorageKey: "animes/upscale_anime/ep_1.mp4",
	}
	if err := upscaleStore.Create(ctx, job); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}
	if job.Status != "queued" {
		t.Errorf("expected status 'queued', got '%s'", job.Status)
	}

	active, err := upscaleStore.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active job, got %d", len(active))
	}
	if active[0].AnimeTitle != "Upscale Anime" {
		t.Errorf("expected anime title 'Upscale Anime', got '%s'", active[0].AnimeTitle)
	}
	if active[0].EpisodeTitle != "Ep 1" {
		t.Errorf("expected episode title 'Ep 1', got '%s'", active[0].EpisodeTitle)
	}
	if active[0].TargetHeight != 1440 {
		t.Errorf("expected target height 1440, got %d", active[0].TargetHeight)
	}
	if active[0].SeasonNumber != 1 {
		t.Errorf("expected season number 1, got %d", active[0].SeasonNumber)
	}
}

func TestSQLiteUpscaleStore_CreateDefaults(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	epStore := store.NewSQLiteEpisodeStore(database)
	upscaleStore := store.NewSQLiteUpscaleStore(database)
	ctx := context.Background()

	anime := seedAnimeWithStorageKey(t, animeStore, epStore)
	epID := anime.Seasons[0].Episodes[0].ID

	job := &model.UpscaleJob{
		EpisodeID:        epID,
		AnimeID:          anime.ID,
		SourceStorageKey: "animes/upscale_anime/ep_1.mp4",
	}
	if err := upscaleStore.Create(ctx, job); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if job.Type != "upscale" {
		t.Errorf("expected default type 'upscale', got '%s'", job.Type)
	}
	if job.TargetHeight != 1080 {
		t.Errorf("expected default target height 1080, got %d", job.TargetHeight)
	}
}

func TestSQLiteUpscaleStore_UpdateStatusAndGetByID(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	epStore := store.NewSQLiteEpisodeStore(database)
	upscaleStore := store.NewSQLiteUpscaleStore(database)
	ctx := context.Background()

	anime := seedAnimeWithStorageKey(t, animeStore, epStore)

	job := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[0].ID,
		AnimeID:          anime.ID,
		SourceStorageKey: "animes/upscale_anime/ep_1.mp4",
	}
	_ = upscaleStore.Create(ctx, job)

	if err := upscaleStore.UpdateStatus(ctx, job.ID.Int64(), "processing", ""); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := upscaleStore.GetByID(ctx, job.ID.Int64())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "processing" {
		t.Errorf("expected status 'processing', got '%s'", got.Status)
	}
}

func TestSQLiteUpscaleStore_UpdateResult(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	epStore := store.NewSQLiteEpisodeStore(database)
	upscaleStore := store.NewSQLiteUpscaleStore(database)
	ctx := context.Background()

	anime := seedAnimeWithStorageKey(t, animeStore, epStore)

	job := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[0].ID,
		AnimeID:          anime.ID,
		SourceStorageKey: "animes/upscale_anime/ep_1.mp4",
	}
	_ = upscaleStore.Create(ctx, job)

	resultKey := "animes/upscale_anime_hq/ep_1.mp4"
	if err := upscaleStore.UpdateResult(ctx, job.ID.Int64(), resultKey); err != nil {
		t.Fatalf("UpdateResult: %v", err)
	}

	got, err := upscaleStore.GetByID(ctx, job.ID.Int64())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ResultStorageKey != resultKey {
		t.Errorf("expected result key '%s', got '%s'", resultKey, got.ResultStorageKey)
	}
}

func TestSQLiteUpscaleStore_Delete(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	epStore := store.NewSQLiteEpisodeStore(database)
	upscaleStore := store.NewSQLiteUpscaleStore(database)
	ctx := context.Background()

	anime := seedAnimeWithStorageKey(t, animeStore, epStore)

	job := &model.UpscaleJob{
		EpisodeID:        anime.Seasons[0].Episodes[0].ID,
		AnimeID:          anime.ID,
		SourceStorageKey: "animes/upscale_anime/ep_1.mp4",
	}
	_ = upscaleStore.Create(ctx, job)

	if err := upscaleStore.Delete(ctx, job.ID.Int64()); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := upscaleStore.GetByID(ctx, job.ID.Int64())
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
