package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"upanime/api/model"
)

func marshalVariants(variants []model.EpisodeVariant) (string, error) {
	if len(variants) == 0 {
		return "", nil
	}
	data, err := json.Marshal(variants)
	if err != nil {
		return "", fmt.Errorf("marshal variants: %w", err)
	}
	return string(data), nil
}

func unmarshalVariants(raw string) []model.EpisodeVariant {
	if raw == "" {
		return nil
	}
	var variants []model.EpisodeVariant
	if err := json.Unmarshal([]byte(raw), &variants); err != nil {
		return nil
	}
	return variants
}

type SQLiteEpisodeStore struct {
	db *sql.DB
}

func NewSQLiteEpisodeStore(db *sql.DB) *SQLiteEpisodeStore {
	return &SQLiteEpisodeStore{db: db}
}

func (s *SQLiteEpisodeStore) GetByID(ctx context.Context, id int64) (*model.Episode, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT e.id, e.season_id, e.anime_id, e.title, e.number, e.url, e.type, e.storage_key, e.upscaled_storage_key, e.upscaled_variants, s.number
		 FROM episodes e
		 JOIN seasons s ON s.id = e.season_id
		 WHERE e.id = ?`,
		id,
	)

	var ep model.Episode
	var epID, seasonID, animeID int64
	var rawVariants string
	err := row.Scan(&epID, &seasonID, &animeID, &ep.Title, &ep.Number, &ep.URL, &ep.Type, &ep.StorageKey, &ep.UpscaledStorageKey, &rawVariants, &ep.SeasonNumber)
	if err != nil {
		return nil, fmt.Errorf("get episode by id: %w", err)
	}
	ep.UpscaledVariants = unmarshalVariants(rawVariants)
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

func (s *SQLiteEpisodeStore) UpdateUpscaledVariants(ctx context.Context, id int64, variants []model.EpisodeVariant) error {
	raw, err := marshalVariants(variants)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE episodes SET upscaled_variants = ? WHERE id = ?",
		raw, id,
	)
	if err != nil {
		return fmt.Errorf("update upscaled variants: %w", err)
	}
	return nil
}
