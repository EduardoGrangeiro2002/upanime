package store

import (
	"context"
	"database/sql"
	"fmt"

	"upanime/api/model"
)

type SQLiteScraperStore struct {
	db *sql.DB
}

func NewSQLiteScraperStore(db *sql.DB) *SQLiteScraperStore {
	return &SQLiteScraperStore{db: db}
}

func (s *SQLiteScraperStore) FindByDomain(ctx context.Context, domain string) (*model.Scraper, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, name, domain, script_path, active, created_at FROM scrapers WHERE domain = ? AND active = 1",
		domain,
	)

	var sc model.Scraper
	var active int
	err := row.Scan(&sc.ID, &sc.Name, &sc.Domain, &sc.ScriptPath, &active, &sc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("find scraper by domain: %w", err)
	}
	sc.Active = active == 1

	return &sc, nil
}
