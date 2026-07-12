package store

import (
	"context"
	"database/sql"
	"fmt"

	"upanime/api/model"
)

type SQLiteEpisodeStore struct {
	db *sql.DB
}

func NewSQLiteEpisodeStore(db *sql.DB) *SQLiteEpisodeStore {
	return &SQLiteEpisodeStore{db: db}
}

func (s *SQLiteEpisodeStore) GetByID(ctx context.Context, id int64) (*model.Episode, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT e.id, e.season_id, e.anime_id, e.title, e.number, e.url, e.type, e.storage_key, e.upscaled_storage_key, s.number
		 FROM episodes e
		 JOIN seasons s ON s.id = e.season_id
		 WHERE e.id = ?`,
		id,
	)

	var ep model.Episode
	var epID, seasonID, animeID int64
	err := row.Scan(&epID, &seasonID, &animeID, &ep.Title, &ep.Number, &ep.URL, &ep.Type, &ep.StorageKey, &ep.UpscaledStorageKey, &ep.SeasonNumber)
	if err != nil {
		return nil, fmt.Errorf("get episode by id: %w", err)
	}
	ep.ID = model.StringID(epID)
	ep.SeasonID = seasonID
	ep.AnimeID = animeID

	return &ep, nil
}

func (s *SQLiteEpisodeStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM episodes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete episode: %w", err)
	}
	return nil
}

func (s *SQLiteEpisodeStore) UpdateStorageKey(ctx context.Context, id int64, key string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE episodes SET storage_key = ? WHERE id = ?",
		key, id,
	)
	if err != nil {
		return fmt.Errorf("update storage key: %w", err)
	}
	return nil
}

func (s *SQLiteEpisodeStore) UpdateUpscaledStorageKey(ctx context.Context, id int64, key string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE episodes SET upscaled_storage_key = ? WHERE id = ?",
		key, id,
	)
	if err != nil {
		return fmt.Errorf("update upscaled storage key: %w", err)
	}
	return nil
}
