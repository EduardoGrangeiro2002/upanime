package store

import (
	"context"
	"database/sql"
	"fmt"

	"upanime/api/model"
)

const datasetSchema = `
CREATE TABLE IF NOT EXISTS dataset_samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source TEXT NOT NULL DEFAULT 'teacher',
	class TEXT NOT NULL,
	frame_key TEXT NOT NULL,
	mask_key TEXT NOT NULL,
	anime_title TEXT NOT NULL DEFAULT '',
	episode TEXT NOT NULL DEFAULT '',
	timestamp_s REAL NOT NULL DEFAULT 0,
	teacher_prob REAL NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'pending',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	reviewed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_dataset_samples_status ON dataset_samples(status);
`

type SQLiteDatasetStore struct {
	db *sql.DB
}

func NewSQLiteDatasetStore(db *sql.DB) (*SQLiteDatasetStore, error) {
	if _, err := db.Exec(datasetSchema); err != nil {
		return nil, fmt.Errorf("migrate dataset schema: %w", err)
	}
	return &SQLiteDatasetStore{db: db}, nil
}

func (s *SQLiteDatasetStore) Create(ctx context.Context, sample *model.DatasetSample) error {
	if sample.Source == "" {
		sample.Source = "teacher"
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO dataset_samples (source, class, frame_key, mask_key, anime_title, episode, timestamp_s, teacher_prob)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.Source, sample.Class, sample.FrameKey, sample.MaskKey,
		sample.AnimeTitle, sample.Episode, sample.TimestampS, sample.TeacherProb,
	)
	if err != nil {
		return fmt.Errorf("insert dataset sample: %w", err)
	}
	id, _ := res.LastInsertId()
	sample.ID = model.StringID(id)
	sample.Status = "pending"
	return nil
}

func (s *SQLiteDatasetStore) Queue(ctx context.Context, limit int) ([]model.DatasetSample, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source, class, frame_key, mask_key, anime_title, episode, timestamp_s, teacher_prob, status, created_at, reviewed_at
		 FROM dataset_samples WHERE status = 'pending' ORDER BY id ASC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list dataset queue: %w", err)
	}
	defer rows.Close()

	samples := []model.DatasetSample{}
	for rows.Next() {
		var sample model.DatasetSample
		var id int64
		if err := rows.Scan(
			&id, &sample.Source, &sample.Class, &sample.FrameKey, &sample.MaskKey,
			&sample.AnimeTitle, &sample.Episode, &sample.TimestampS, &sample.TeacherProb,
			&sample.Status, &sample.CreatedAt, &sample.ReviewedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dataset sample: %w", err)
		}
		sample.ID = model.StringID(id)
		samples = append(samples, sample)
	}
	return samples, nil
}

func (s *SQLiteDatasetStore) SetVerdict(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE dataset_samples SET status = ?, reviewed_at = datetime('now') WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update dataset verdict: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteDatasetStore) Stats(ctx context.Context) ([]model.DatasetClassStat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT class, status, COUNT(*) FROM dataset_samples GROUP BY class, status ORDER BY class, status`)
	if err != nil {
		return nil, fmt.Errorf("dataset stats: %w", err)
	}
	defer rows.Close()

	stats := []model.DatasetClassStat{}
	for rows.Next() {
		var stat model.DatasetClassStat
		if err := rows.Scan(&stat.Class, &stat.Status, &stat.Count); err != nil {
			return nil, fmt.Errorf("scan dataset stat: %w", err)
		}
		stats = append(stats, stat)
	}
	return stats, nil
}
