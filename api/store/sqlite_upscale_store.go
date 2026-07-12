package store

import (
	"context"
	"database/sql"
	"fmt"

	"upanime/api/model"
)

type SQLiteUpscaleStore struct {
	db *sql.DB
}

func NewSQLiteUpscaleStore(db *sql.DB) *SQLiteUpscaleStore {
	return &SQLiteUpscaleStore{db: db}
}

func (s *SQLiteUpscaleStore) Create(ctx context.Context, job *model.UpscaleJob) error {
	jobType := job.Type
	if jobType == "" {
		jobType = "upscale"
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO upscale_jobs (episode_id, anime_id, source_storage_key, result_storage_key, type, target_height, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'queued')`,
		job.EpisodeID.Int64(), job.AnimeID.Int64(), job.SourceStorageKey, job.ResultStorageKey, jobType, defaultTargetHeight(job.TargetHeight),
	)
	if err != nil {
		return fmt.Errorf("insert upscale job: %w", err)
	}

	id, _ := res.LastInsertId()
	job.ID = model.StringID(id)
	job.Status = "queued"
	job.Type = jobType
	job.TargetHeight = defaultTargetHeight(job.TargetHeight)
	return nil
}

func (s *SQLiteUpscaleStore) ListActive(ctx context.Context) ([]model.UpscaleJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.episode_id, u.anime_id, u.type, u.target_height, u.source_storage_key, u.result_storage_key,
			u.runpod_job_id, u.status, u.error, u.created_at, u.updated_at,
			e.title, e.number, s.number,
			a.title, a.image_url
		FROM upscale_jobs u
		JOIN episodes e ON e.id = u.episode_id
		JOIN seasons s ON s.id = e.season_id
		JOIN animes a ON a.id = u.anime_id
		WHERE u.status NOT IN ('completed', 'failed')
			OR u.updated_at >= datetime('now', '-24 hours')
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list active upscale jobs: %w", err)
	}
	defer rows.Close()

	var jobs []model.UpscaleJob
	for rows.Next() {
		var j model.UpscaleJob
		var id, episodeID, animeID int64
		if err := rows.Scan(
			&id, &episodeID, &animeID, &j.Type, &j.TargetHeight, &j.SourceStorageKey, &j.ResultStorageKey,
			&j.RunPodJobID, &j.Status, &j.Error, &j.CreatedAt, &j.UpdatedAt,
			&j.EpisodeTitle, &j.EpisodeNumber, &j.SeasonNumber,
			&j.AnimeTitle, &j.AnimeImageURL,
		); err != nil {
			return nil, fmt.Errorf("scan upscale job: %w", err)
		}
		j.ID = model.StringID(id)
		j.EpisodeID = model.StringID(episodeID)
		j.AnimeID = model.StringID(animeID)
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (s *SQLiteUpscaleStore) ListProcessing(ctx context.Context) ([]model.UpscaleJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.episode_id, u.anime_id, u.type, u.target_height, u.source_storage_key, u.result_storage_key,
			u.runpod_job_id, u.status, u.error, u.created_at, u.updated_at,
			e.title, e.number, s.number,
			a.title, a.image_url
		FROM upscale_jobs u
		JOIN episodes e ON e.id = u.episode_id
		JOIN seasons s ON s.id = e.season_id
		JOIN animes a ON a.id = u.anime_id
		WHERE u.status IN ('queued', 'processing')
			AND u.runpod_job_id != ''
		ORDER BY u.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list processing upscale jobs: %w", err)
	}
	defer rows.Close()

	var jobs []model.UpscaleJob
	for rows.Next() {
		var j model.UpscaleJob
		var id, episodeID, animeID int64
		if err := rows.Scan(
			&id, &episodeID, &animeID, &j.Type, &j.TargetHeight, &j.SourceStorageKey, &j.ResultStorageKey,
			&j.RunPodJobID, &j.Status, &j.Error, &j.CreatedAt, &j.UpdatedAt,
			&j.EpisodeTitle, &j.EpisodeNumber, &j.SeasonNumber,
			&j.AnimeTitle, &j.AnimeImageURL,
		); err != nil {
			return nil, fmt.Errorf("scan processing upscale job: %w", err)
		}
		j.ID = model.StringID(id)
		j.EpisodeID = model.StringID(episodeID)
		j.AnimeID = model.StringID(animeID)
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (s *SQLiteUpscaleStore) GetByID(ctx context.Context, id int64) (*model.UpscaleJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.episode_id, u.anime_id, u.type, u.target_height, u.source_storage_key, u.result_storage_key,
			u.runpod_job_id, u.status, u.error, u.created_at, u.updated_at,
			e.title, e.number, s.number,
			a.title, a.image_url
		FROM upscale_jobs u
		JOIN episodes e ON e.id = u.episode_id
		JOIN seasons s ON s.id = e.season_id
		JOIN animes a ON a.id = u.anime_id
		WHERE u.id = ?
	`, id)

	var j model.UpscaleJob
	var jID, episodeID, animeID int64
	err := row.Scan(
		&jID, &episodeID, &animeID, &j.Type, &j.TargetHeight, &j.SourceStorageKey, &j.ResultStorageKey,
		&j.RunPodJobID, &j.Status, &j.Error, &j.CreatedAt, &j.UpdatedAt,
		&j.EpisodeTitle, &j.EpisodeNumber, &j.SeasonNumber,
		&j.AnimeTitle, &j.AnimeImageURL,
	)
	if err != nil {
		return nil, fmt.Errorf("get upscale job by id: %w", err)
	}
	j.ID = model.StringID(jID)
	j.EpisodeID = model.StringID(episodeID)
	j.AnimeID = model.StringID(animeID)

	return &j, nil
}

func (s *SQLiteUpscaleStore) UpdateStatus(ctx context.Context, id int64, status string, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE upscale_jobs SET status = ?, error = ?, updated_at = datetime('now') WHERE id = ?",
		status, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update upscale job status: %w", err)
	}
	return nil
}

func (s *SQLiteUpscaleStore) UpdateResult(ctx context.Context, id int64, resultKey string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE upscale_jobs SET result_storage_key = ?, updated_at = datetime('now') WHERE id = ?",
		resultKey, id,
	)
	if err != nil {
		return fmt.Errorf("update upscale result key: %w", err)
	}
	return nil
}

func (s *SQLiteUpscaleStore) UpdateRunPodJobID(ctx context.Context, id int64, runpodJobID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE upscale_jobs SET runpod_job_id = ?, updated_at = datetime('now') WHERE id = ?",
		runpodJobID, id,
	)
	if err != nil {
		return fmt.Errorf("update runpod job id: %w", err)
	}
	return nil
}

func (s *SQLiteUpscaleStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM upscale_jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete upscale job: %w", err)
	}
	return nil
}

func defaultTargetHeight(height int) int {
	if height == 0 {
		return 1080
	}
	return height
}
