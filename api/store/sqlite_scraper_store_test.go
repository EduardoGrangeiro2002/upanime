package store_test

import (
	"context"
	"testing"

	"upanime/api/store"
	"upanime/api/testutil"
)

func TestSQLiteScraperStore_FindByDomain(t *testing.T) {
	db := testutil.NewTestDB(t)
	s := store.NewSQLiteScraperStore(db)
	ctx := context.Background()

	scraper, err := s.FindByDomain(ctx, "animesonlinecc.to")
	if err != nil {
		t.Fatalf("FindByDomain: %v", err)
	}

	if scraper.Name != "animesonlinecc" {
		t.Errorf("expected name 'animesonlinecc', got '%s'", scraper.Name)
	}
	if scraper.ScriptPath != "sites/animesonlinecc.py" {
		t.Errorf("expected script_path 'sites/animesonlinecc.py', got '%s'", scraper.ScriptPath)
	}
}

func TestSQLiteScraperStore_FindByDomain_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	s := store.NewSQLiteScraperStore(db)
	ctx := context.Background()

	_, err := s.FindByDomain(ctx, "unknown.com")
	if err == nil {
		t.Fatal("expected error for unknown domain")
	}
}
