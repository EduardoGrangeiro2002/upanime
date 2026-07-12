package store_test

import (
	"context"
	"testing"

	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func TestSQLiteAnimeStore_CreateAndFindByURL(t *testing.T) {
	db := testutil.NewTestDB(t)
	s := store.NewSQLiteAnimeStore(db)
	ctx := context.Background()

	anime := &model.Anime{
		Title:       "Test Anime",
		URL:         "https://animesonlinecc.to/anime/test",
		ImageURL:    "https://example.com/img.jpg",
		Description: "A test anime",
		ScraperID:   1,
		Seasons: []model.Season{
			{
				Number: 1,
				Label:  "Season 1",
				Type:   "episode",
				Episodes: []model.Episode{
					{Title: "Ep 1", Number: "1", URL: "https://animesonlinecc.to/episodio/test-1", Type: "episode"},
					{Title: "Ep 2", Number: "2", URL: "https://animesonlinecc.to/episodio/test-2", Type: "episode"},
				},
			},
		},
	}

	err := s.Create(ctx, anime)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if anime.ID.Int64() == 0 {
		t.Fatal("expected anime ID to be set")
	}
	if anime.Seasons[0].ID == 0 {
		t.Fatal("expected season ID to be set")
	}
	if anime.Seasons[0].Episodes[0].ID.Int64() == 0 {
		t.Fatal("expected episode ID to be set")
	}

	found, err := s.FindByURL(ctx, "https://animesonlinecc.to/anime/test")
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}

	if found.Title != "Test Anime" {
		t.Errorf("expected title 'Test Anime', got '%s'", found.Title)
	}
	if len(found.Seasons) != 1 {
		t.Fatalf("expected 1 season, got %d", len(found.Seasons))
	}
	if len(found.Seasons[0].Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(found.Seasons[0].Episodes))
	}
	if found.Seasons[0].Episodes[0].SeasonNumber != 1 {
		t.Errorf("expected seasonNumber 1, got %d", found.Seasons[0].Episodes[0].SeasonNumber)
	}
}

func TestSQLiteAnimeStore_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	s := store.NewSQLiteAnimeStore(db)
	ctx := context.Background()

	anime := &model.Anime{
		Title:     "ByID Anime",
		URL:       "https://animesonlinecc.to/anime/byid",
		ScraperID: 1,
		Seasons:   []model.Season{{Number: 1, Type: "episode"}},
	}
	_ = s.Create(ctx, anime)

	found, err := s.GetByID(ctx, anime.ID.Int64())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.Title != "ByID Anime" {
		t.Errorf("expected 'ByID Anime', got '%s'", found.Title)
	}
}
