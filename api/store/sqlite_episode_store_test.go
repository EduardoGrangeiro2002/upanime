package store_test

import (
	"testing"

	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func TestSQLiteEpisodeStore_UpdateUpscaledStorageKey(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	episodeStore := store.NewSQLiteEpisodeStore(db)

	anime := &model.Anime{
		Title:     "Episode Store Anime",
		URL:       "https://example.com/episode-store",
		ScraperID: 1,
		Seasons: []model.Season{
			{
				Number: 1,
				Label:  "Season 1",
				Type:   "episode",
				Episodes: []model.Episode{
					{Title: "Ep 1", Number: "1", URL: "https://example.com/ep1", Type: "episode"},
				},
			},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	episodeID := anime.Seasons[0].Episodes[0].ID.Int64()
	if err := episodeStore.UpdateUpscaledStorageKey(t.Context(), episodeID, "animes/test/ep1_upscaled.mp4"); err != nil {
		t.Fatalf("update upscaled storage key: %v", err)
	}

	episode, err := episodeStore.GetByID(t.Context(), episodeID)
	if err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if episode.UpscaledStorageKey != "animes/test/ep1_upscaled.mp4" {
		t.Fatalf("expected upscaled storage key to persist, got %q", episode.UpscaledStorageKey)
	}
}

func TestSQLiteEpisodeStore_UpdateUpscaledVariants(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	episodeStore := store.NewSQLiteEpisodeStore(db)

	anime := &model.Anime{
		Title:     "Variants Anime",
		URL:       "https://example.com/variants",
		ScraperID: 1,
		Seasons: []model.Season{
			{
				Number: 1,
				Label:  "Season 1",
				Type:   "episode",
				Episodes: []model.Episode{
					{Title: "Ep 1", Number: "1", URL: "https://example.com/var-ep1", Type: "episode"},
				},
			},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	episodeID := anime.Seasons[0].Episodes[0].ID.Int64()
	variants := []model.EpisodeVariant{
		{Height: 2160, StorageKey: "animes/test/ep1_upscaled.mp4"},
		{Height: 1440, StorageKey: "animes/test/ep1_upscaled_1440p.mp4"},
		{Height: 1080, StorageKey: "animes/test/ep1_upscaled_1080p.mp4"},
	}
	if err := episodeStore.UpdateUpscaledVariants(t.Context(), episodeID, variants); err != nil {
		t.Fatalf("update upscaled variants: %v", err)
	}

	episode, err := episodeStore.GetByID(t.Context(), episodeID)
	if err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if len(episode.UpscaledVariants) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(episode.UpscaledVariants))
	}
	if episode.UpscaledVariants[1].Height != 1440 || episode.UpscaledVariants[1].StorageKey != "animes/test/ep1_upscaled_1440p.mp4" {
		t.Fatalf("unexpected variant: %+v", episode.UpscaledVariants[1])
	}

	loaded, err := animeStore.GetByID(t.Context(), anime.ID.Int64())
	if err != nil {
		t.Fatalf("get anime: %v", err)
	}
	if len(loaded.Seasons[0].Episodes[0].UpscaledVariants) != 3 {
		t.Fatalf("expected variants via anime store, got %+v", loaded.Seasons[0].Episodes[0].UpscaledVariants)
	}
}

func TestSQLiteEpisodeStore_UpdateUpscaledVariantsEmptyClearsColumn(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	episodeStore := store.NewSQLiteEpisodeStore(db)

	anime := &model.Anime{
		Title:     "Variants Clear Anime",
		URL:       "https://example.com/variants-clear",
		ScraperID: 1,
		Seasons: []model.Season{
			{
				Number: 1,
				Label:  "Season 1",
				Type:   "episode",
				Episodes: []model.Episode{
					{Title: "Ep 1", Number: "1", URL: "https://example.com/varclear-ep1", Type: "episode"},
				},
			},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}

	episodeID := anime.Seasons[0].Episodes[0].ID.Int64()
	if err := episodeStore.UpdateUpscaledVariants(t.Context(), episodeID, nil); err != nil {
		t.Fatalf("update with nil variants: %v", err)
	}

	episode, err := episodeStore.GetByID(t.Context(), episodeID)
	if err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if episode.UpscaledVariants != nil {
		t.Fatalf("expected nil variants, got %+v", episode.UpscaledVariants)
	}
}
