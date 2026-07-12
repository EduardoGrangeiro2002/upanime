package store

import (
	"context"
	"database/sql"
	"fmt"

	"upanime/api/model"
)

type SQLiteDownloadStore struct {
	db *sql.DB
}

func NewSQLiteDownloadStore(db *sql.DB) *SQLiteDownloadStore {
	return &SQLiteDownloadStore{db: db}
}

func (s *SQLiteDownloadStore) Create(ctx context.Context, downloads []model.Download) ([]model.Download, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var result []model.Download
	for _, d := range downloads {
		res, err := tx.ExecContext(ctx,
			"INSERT INTO downloads (episode_id, anime_id, status) VALUES (?, ?, 'queued')",
			d.EpisodeID.Int64(), d.AnimeID.Int64(),
		)
		if err != nil {
			return nil, fmt.Errorf("insert download: %w", err)
		}

		id, _ := res.LastInsertId()
		d.ID = model.StringID(id)
		d.Status = "queued"
		result = append(result, d)
	}

	return result, tx.Commit()
}

func (s *SQLiteDownloadStore) ListActive(ctx context.Context) ([]model.Download, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.episode_id, d.anime_id, d.status, d.progress, d.speed, d.eta, d.error, d.dest_path, d.created_at, d.updated_at,
			e.title, e.number, s.number,
			a.title, a.image_url
		FROM downloads d
		JOIN episodes e ON e.id = d.episode_id
		JOIN seasons s ON s.id = e.season_id
		JOIN animes a ON a.id = d.anime_id
		WHERE d.status NOT IN ('completed', 'failed')
			OR d.updated_at >= datetime('now', '-30 seconds')
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list active downloads: %w", err)
	}
	defer rows.Close()

	var downloads []model.Download
	for rows.Next() {
		var d model.Download
		var id, episodeID, animeID int64
		if err := rows.Scan(
			&id, &episodeID, &animeID, &d.Status, &d.Progress, &d.Speed, &d.ETA, &d.Error, &d.DestPath, &d.CreatedAt, &d.UpdatedAt,
			&d.EpisodeTitle, &d.EpisodeNumber, &d.SeasonNumber,
			&d.AnimeTitle, &d.AnimeImageURL,
		); err != nil {
			return nil, fmt.Errorf("scan download: %w", err)
		}
		d.ID = model.StringID(id)
		d.EpisodeID = model.StringID(episodeID)
		d.AnimeID = model.StringID(animeID)
		downloads = append(downloads, d)
	}

	return downloads, nil
}

func (s *SQLiteDownloadStore) GetByID(ctx context.Context, id int64) (*model.Download, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT d.id, d.episode_id, d.anime_id, d.status, d.progress, d.speed, d.eta, d.error, d.dest_path, d.created_at, d.updated_at,
			e.title, e.number, s.number,
			a.title, a.image_url
		FROM downloads d
		JOIN episodes e ON e.id = d.episode_id
		JOIN seasons s ON s.id = e.season_id
		JOIN animes a ON a.id = d.anime_id
		WHERE d.id = ?
	`, id)

	var d model.Download
	var dlID, episodeID, animeID int64
	err := row.Scan(
		&dlID, &episodeID, &animeID, &d.Status, &d.Progress, &d.Speed, &d.ETA, &d.Error, &d.DestPath, &d.CreatedAt, &d.UpdatedAt,
		&d.EpisodeTitle, &d.EpisodeNumber, &d.SeasonNumber,
		&d.AnimeTitle, &d.AnimeImageURL,
	)
	if err != nil {
		return nil, fmt.Errorf("get download by id: %w", err)
	}
	d.ID = model.StringID(dlID)
	d.EpisodeID = model.StringID(episodeID)
	d.AnimeID = model.StringID(animeID)

	return &d, nil
}

func (s *SQLiteDownloadStore) UpdateStatus(ctx context.Context, id int64, status string, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE downloads SET status = ?, error = ?, updated_at = datetime('now') WHERE id = ?",
		status, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update download status: %w", err)
	}

	return nil
}

func (s *SQLiteDownloadStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM downloads WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete download: %w", err)
	}

	return nil
}
