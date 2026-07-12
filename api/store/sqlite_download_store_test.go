package store_test

import (
	"context"
	"testing"

	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func TestSQLiteDownloadStore_CreateAndListActive(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	dlStore := store.NewSQLiteDownloadStore(database)
	ctx := context.Background()

	anime := &model.Anime{
		Title:     "DL Anime",
		URL:       "https://animesonlinecc.to/anime/dl",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/ep1", Type: "episode"},
			}},
		},
	}
	if err := animeStore.Create(ctx, anime); err != nil {
		t.Fatalf("seed anime: %v", err)
	}

	epID := anime.Seasons[0].Episodes[0].ID

	downloads, err := dlStore.Create(ctx, []model.Download{
		{EpisodeID: epID, AnimeID: anime.ID},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(downloads) != 1 {
		t.Fatalf("expected 1 download, got %d", len(downloads))
	}
	if downloads[0].Status != "queued" {
		t.Errorf("expected status 'queued', got '%s'", downloads[0].Status)
	}

	active, err := dlStore.ListActive(ctx)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active download, got %d", len(active))
	}
	if active[0].AnimeTitle != "DL Anime" {
		t.Errorf("expected anime title 'DL Anime', got '%s'", active[0].AnimeTitle)
	}
	if active[0].SeasonNumber != 1 {
		t.Errorf("expected season number 1, got %d", active[0].SeasonNumber)
	}
}

func TestSQLiteDownloadStore_UpdateStatusAndGetByID(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	dlStore := store.NewSQLiteDownloadStore(database)
	ctx := context.Background()

	anime := &model.Anime{
		Title:     "Status Anime",
		URL:       "https://animesonlinecc.to/anime/status",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/ep-status-1", Type: "episode"},
			}},
		},
	}
	_ = animeStore.Create(ctx, anime)

	downloads, _ := dlStore.Create(ctx, []model.Download{
		{EpisodeID: anime.Seasons[0].Episodes[0].ID, AnimeID: anime.ID},
	})

	err := dlStore.UpdateStatus(ctx, downloads[0].ID.Int64(), "downloading", "")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	d, err := dlStore.GetByID(ctx, downloads[0].ID.Int64())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if d.Status != "downloading" {
		t.Errorf("expected status 'downloading', got '%s'", d.Status)
	}
}

func TestSQLiteDownloadStore_Delete(t *testing.T) {
	database := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(database)
	dlStore := store.NewSQLiteDownloadStore(database)
	ctx := context.Background()

	anime := &model.Anime{
		Title:     "Del Anime",
		URL:       "https://animesonlinecc.to/anime/del",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/ep-del-1", Type: "episode"},
			}},
		},
	}
	_ = animeStore.Create(ctx, anime)

	downloads, _ := dlStore.Create(ctx, []model.Download{
		{EpisodeID: anime.Seasons[0].Episodes[0].ID, AnimeID: anime.ID},
	})

	err := dlStore.Delete(ctx, downloads[0].ID.Int64())
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = dlStore.GetByID(ctx, downloads[0].ID.Int64())
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
