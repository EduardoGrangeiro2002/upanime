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
