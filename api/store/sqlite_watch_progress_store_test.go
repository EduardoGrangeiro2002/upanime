package store_test

import (
	"database/sql"
	"errors"
	"testing"

	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func seedProgressAnime(t *testing.T, db *sql.DB) *model.Anime {
	t.Helper()
	animes := store.NewSQLiteAnimeStore(db)
	anime := &model.Anime{
		Title:     "Progress Anime",
		URL:       "https://animesonlinecc.to/anime/progress-anime",
		ImageURL:  "https://example.com/progress.jpg",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Type: "episode", Episodes: []model.Episode{
				{Title: "Ep 1", Number: "1", URL: "https://example.com/wp-1", Type: "episode"},
				{Title: "Ep 2", Number: "2", URL: "https://example.com/wp-2", Type: "episode"},
				{Title: "Ep 3", Number: "3", URL: "https://example.com/wp-3", Type: "episode"},
			}},
		},
	}
	if err := animes.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}
	return anime
}

func setUpdatedAt(t *testing.T, db *sql.DB, email string, episodeID int64, updatedAt string) {
	t.Helper()
	if _, err := db.Exec("UPDATE watch_progress SET updated_at = ? WHERE user_email = ? AND episode_id = ?", updatedAt, email, episodeID); err != nil {
		t.Fatalf("set updated_at: %v", err)
	}
}

func TestWatchProgressStore_UpsertAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	anime := seedProgressAnime(t, db)
	s := store.NewSQLiteWatchProgressStore(db)
	epID := anime.Seasons[0].Episodes[0].ID.Int64()

	if err := s.Upsert(t.Context(), "eu@teste.dev", epID, 123.4, 1420); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	p, err := s.Get(t.Context(), "eu@teste.dev", epID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if p.Position != 123.4 || p.Duration != 1420 {
		t.Fatalf("expected 123.4/1420, got %v/%v", p.Position, p.Duration)
	}

	if err := s.Upsert(t.Context(), "eu@teste.dev", epID, 500, 1420); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	p, err = s.Get(t.Context(), "eu@teste.dev", epID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if p.Position != 500 {
		t.Fatalf("expected position updated to 500, got %v", p.Position)
	}
}

func TestWatchProgressStore_GetMissing(t *testing.T) {
	db := testutil.NewTestDB(t)
	s := store.NewSQLiteWatchProgressStore(db)

	_, err := s.Get(t.Context(), "eu@teste.dev", 9999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestWatchProgressStore_UpsertUnknownEpisodeFails(t *testing.T) {
	db := testutil.NewTestDB(t)
	s := store.NewSQLiteWatchProgressStore(db)

	if err := s.Upsert(t.Context(), "eu@teste.dev", 9999, 10, 100); err == nil {
		t.Fatal("expected foreign key error for unknown episode")
	}
}

func TestWatchProgressStore_ListInProgress(t *testing.T) {
	db := testutil.NewTestDB(t)
	anime := seedProgressAnime(t, db)
	s := store.NewSQLiteWatchProgressStore(db)
	eps := anime.Seasons[0].Episodes

	if err := s.Upsert(t.Context(), "eu@teste.dev", eps[0].ID.Int64(), 100, 1400); err != nil {
		t.Fatalf("upsert ep1: %v", err)
	}
	if err := s.Upsert(t.Context(), "eu@teste.dev", eps[1].ID.Int64(), 1390, 1400); err != nil {
		t.Fatalf("upsert ep2: %v", err)
	}
	if err := s.Upsert(t.Context(), "eu@teste.dev", eps[2].ID.Int64(), 3, 1400); err != nil {
		t.Fatalf("upsert ep3: %v", err)
	}
	if err := s.Upsert(t.Context(), "outra@teste.dev", eps[2].ID.Int64(), 200, 1400); err != nil {
		t.Fatalf("upsert other user: %v", err)
	}

	items, err := s.ListInProgress(t.Context(), "eu@teste.dev", 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (completed and <5s excluded), got %d", len(items))
	}
	item := items[0]
	if item.EpisodeID != eps[0].ID {
		t.Errorf("expected episode %v, got %v", eps[0].ID, item.EpisodeID)
	}
	if item.AnimeTitle != "Progress Anime" {
		t.Errorf("expected anime title joined, got '%s'", item.AnimeTitle)
	}
	if item.EpisodeNumber != "1" || item.SeasonNumber != 1 {
		t.Errorf("expected episode 1 season 1, got %s/%d", item.EpisodeNumber, item.SeasonNumber)
	}
	if item.Position != 100 || item.Duration != 1400 {
		t.Errorf("expected 100/1400, got %v/%v", item.Position, item.Duration)
	}
}

func TestWatchProgressStore_ListOrderAndLimit(t *testing.T) {
	db := testutil.NewTestDB(t)
	anime := seedProgressAnime(t, db)
	s := store.NewSQLiteWatchProgressStore(db)
	eps := anime.Seasons[0].Episodes

	for _, ep := range eps {
		if err := s.Upsert(t.Context(), "eu@teste.dev", ep.ID.Int64(), 100, 1400); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	setUpdatedAt(t, db, "eu@teste.dev", eps[0].ID.Int64(), "2026-07-18 10:00:00.000")
	setUpdatedAt(t, db, "eu@teste.dev", eps[1].ID.Int64(), "2026-07-19 10:00:00.000")
	setUpdatedAt(t, db, "eu@teste.dev", eps[2].ID.Int64(), "2026-07-17 10:00:00.000")

	items, err := s.ListInProgress(t.Context(), "eu@teste.dev", 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].EpisodeID != eps[1].ID || items[1].EpisodeID != eps[0].ID || items[2].EpisodeID != eps[2].ID {
		t.Fatalf("expected order ep2, ep1, ep3, got %v, %v, %v", items[0].EpisodeID, items[1].EpisodeID, items[2].EpisodeID)
	}

	limited, err := s.ListInProgress(t.Context(), "eu@teste.dev", 1)
	if err != nil {
		t.Fatalf("list limited: %v", err)
	}
	if len(limited) != 1 || limited[0].EpisodeID != eps[1].ID {
		t.Fatalf("expected only most recent item, got %v", limited)
	}
}
