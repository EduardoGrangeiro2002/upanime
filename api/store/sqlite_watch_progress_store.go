package store

import (
	"context"
	"database/sql"
	"fmt"

	"upanime/api/model"
)

type SQLiteWatchProgressStore struct {
	db *sql.DB
}

func NewSQLiteWatchProgressStore(db *sql.DB) *SQLiteWatchProgressStore {
	return &SQLiteWatchProgressStore{db: db}
}

func (s *SQLiteWatchProgressStore) Upsert(ctx context.Context, email string, episodeID int64, position, duration float64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watch_progress (user_email, episode_id, position_seconds, duration_seconds, updated_at)
		VALUES (?, ?, ?, ?, strftime('%Y-%m-%d %H:%M:%f', 'now'))
		ON CONFLICT(user_email, episode_id) DO UPDATE SET
			position_seconds = excluded.position_seconds,
			duration_seconds = excluded.duration_seconds,
			updated_at = excluded.updated_at
	`, email, episodeID, position, duration)
	if err != nil {
		return fmt.Errorf("upsert watch progress: %w", err)
	}

	return nil
}

func (s *SQLiteWatchProgressStore) Get(ctx context.Context, email string, episodeID int64) (*model.WatchProgress, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT episode_id, position_seconds, duration_seconds, updated_at
		FROM watch_progress
		WHERE user_email = ? AND episode_id = ?
	`, email, episodeID)

	var p model.WatchProgress
	var epID int64
	if err := row.Scan(&epID, &p.Position, &p.Duration, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("get watch progress: %w", err)
	}
	p.EpisodeID = model.StringID(epID)

	return &p, nil
}

func (s *SQLiteWatchProgressStore) ListInProgress(ctx context.Context, email string, limit int) ([]model.WatchProgress, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT wp.episode_id, wp.position_seconds, wp.duration_seconds, wp.updated_at,
			e.title, e.number, s.number,
			a.id, a.title, a.image_url
		FROM watch_progress wp
		JOIN episodes e ON e.id = wp.episode_id
		JOIN seasons s ON s.id = e.season_id
		JOIN animes a ON a.id = e.anime_id
		WHERE wp.user_email = ?
			AND wp.position_seconds > 5
			AND (wp.duration_seconds <= 0 OR wp.position_seconds < wp.duration_seconds * 0.95)
		ORDER BY wp.updated_at DESC
		LIMIT ?
	`, email, limit)
	if err != nil {
		return nil, fmt.Errorf("list watch progress: %w", err)
	}
	defer rows.Close()

	var items []model.WatchProgress
	for rows.Next() {
		var p model.WatchProgress
		var epID, animeID int64
		if err := rows.Scan(
			&epID, &p.Position, &p.Duration, &p.UpdatedAt,
			&p.EpisodeTitle, &p.EpisodeNumber, &p.SeasonNumber,
			&animeID, &p.AnimeTitle, &p.AnimeImageURL,
		); err != nil {
			return nil, fmt.Errorf("scan watch progress: %w", err)
		}
		p.EpisodeID = model.StringID(epID)
		p.AnimeID = model.StringID(animeID)
		items = append(items, p)
	}

	return items, nil
}
