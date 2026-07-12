package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"upanime/api/model"
)

type SQLiteAnimeStore struct {
	db *sql.DB
}

func NewSQLiteAnimeStore(db *sql.DB) *SQLiteAnimeStore {
	return &SQLiteAnimeStore{db: db}
}

func (s *SQLiteAnimeStore) FindByURL(ctx context.Context, url string) (*model.Anime, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, title, url, image_url, description, cover_path, genres, scraper_id, created_at, updated_at FROM animes WHERE url = ?",
		url,
	)

	var a model.Anime
	var id, scraperID int64
	var genresJSON string
	err := row.Scan(&id, &a.Title, &a.URL, &a.ImageURL, &a.Description, &a.CoverPath, &genresJSON, &scraperID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("find anime by url: %w", err)
	}
	a.ID = model.StringID(id)
	a.ScraperID = scraperID
	a.Genres = genresFromJSON(genresJSON)

	seasons, err := s.loadSeasons(ctx, id)
	if err != nil {
		return nil, err
	}
	a.Seasons = seasons

	return &a, nil
}

func (s *SQLiteAnimeStore) Create(ctx context.Context, anime *model.Anime) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		"INSERT INTO animes (title, url, image_url, description, genres, scraper_id) VALUES (?, ?, ?, ?, ?, ?)",
		anime.Title, anime.URL, anime.ImageURL, anime.Description, genresToJSON(anime.Genres), anime.ScraperID,
	)
	if err != nil {
		return fmt.Errorf("insert anime: %w", err)
	}

	animeID, _ := res.LastInsertId()
	anime.ID = model.StringID(animeID)

	for i := range anime.Seasons {
		season := &anime.Seasons[i]
		season.AnimeID = animeID

		sRes, err := tx.ExecContext(ctx,
			"INSERT INTO seasons (anime_id, number, label, type) VALUES (?, ?, ?, ?)",
			animeID, season.Number, season.Label, season.Type,
		)
		if err != nil {
			return fmt.Errorf("insert season: %w", err)
		}

		seasonID, _ := sRes.LastInsertId()
		season.ID = seasonID

		for j := range season.Episodes {
			ep := &season.Episodes[j]
			ep.SeasonID = seasonID
			ep.AnimeID = animeID
			ep.SeasonNumber = season.Number

			eRes, err := tx.ExecContext(ctx,
				"INSERT INTO episodes (season_id, anime_id, title, number, url, type) VALUES (?, ?, ?, ?, ?, ?)",
				seasonID, animeID, ep.Title, ep.Number, ep.URL, ep.Type,
			)
			if err != nil {
				return fmt.Errorf("insert episode: %w", err)
			}

			epID, _ := eRes.LastInsertId()
			ep.ID = model.StringID(epID)
		}
	}

	return tx.Commit()
}

func (s *SQLiteAnimeStore) GetByID(ctx context.Context, id int64) (*model.Anime, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, title, url, image_url, description, cover_path, genres, scraper_id, created_at, updated_at FROM animes WHERE id = ?",
		id,
	)

	var a model.Anime
	var animeID, scraperID int64
	var genresJSON string
	err := row.Scan(&animeID, &a.Title, &a.URL, &a.ImageURL, &a.Description, &a.CoverPath, &genresJSON, &scraperID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get anime by id: %w", err)
	}
	a.ID = model.StringID(animeID)
	a.ScraperID = scraperID
	a.Genres = genresFromJSON(genresJSON)

	seasons, err := s.loadSeasons(ctx, animeID)
	if err != nil {
		return nil, err
	}
	a.Seasons = seasons

	return &a, nil
}

func (s *SQLiteAnimeStore) List(ctx context.Context) ([]model.Anime, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, url, image_url, description, cover_path, genres, scraper_id, created_at, updated_at FROM animes ORDER BY title",
	)
	if err != nil {
		return nil, fmt.Errorf("list animes: %w", err)
	}
	defer rows.Close()

	var animes []model.Anime
	for rows.Next() {
		var a model.Anime
		var id, scraperID int64
		var genresJSON string
		if err := rows.Scan(&id, &a.Title, &a.URL, &a.ImageURL, &a.Description, &a.CoverPath, &genresJSON, &scraperID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan anime: %w", err)
		}
		a.ID = model.StringID(id)
		a.ScraperID = scraperID
		a.Genres = genresFromJSON(genresJSON)

		seasons, err := s.loadSeasons(ctx, id)
		if err != nil {
			return nil, err
		}
		a.Seasons = seasons

		animes = append(animes, a)
	}

	return animes, nil
}

func (s *SQLiteAnimeStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM animes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete anime: %w", err)
	}
	return nil
}

func (s *SQLiteAnimeStore) UpdateCoverPath(ctx context.Context, id int64, path string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE animes SET cover_path = ?, updated_at = datetime('now') WHERE id = ?",
		path, id,
	)
	if err != nil {
		return fmt.Errorf("update cover path: %w", err)
	}
	return nil
}

func (s *SQLiteAnimeStore) UpdateGenres(ctx context.Context, id int64, genres []string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE animes SET genres = ?, updated_at = datetime('now') WHERE id = ?",
		genresToJSON(genres), id,
	)
	if err != nil {
		return fmt.Errorf("update genres: %w", err)
	}
	return nil
}

func (s *SQLiteAnimeStore) UpdateEpisodeNumber(ctx context.Context, episodeID int64, number string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE episodes SET number = ? WHERE id = ?",
		number, episodeID,
	)
	if err != nil {
		return fmt.Errorf("update episode number: %w", err)
	}
	return nil
}

func (s *SQLiteAnimeStore) AddEpisode(ctx context.Context, animeID int64, seasonNumber int, ep *model.Episode) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var seasonID int64
	row := tx.QueryRowContext(ctx,
		"SELECT id FROM seasons WHERE anime_id = ? AND number = ? AND type = 'episode'",
		animeID, seasonNumber,
	)
	err = row.Scan(&seasonID)
	if err == sql.ErrNoRows {
		res, insertErr := tx.ExecContext(ctx,
			"INSERT INTO seasons (anime_id, number, label, type) VALUES (?, ?, ?, 'episode')",
			animeID, seasonNumber, fmt.Sprintf("Temporada %d", seasonNumber),
		)
		if insertErr != nil {
			return fmt.Errorf("insert season: %w", insertErr)
		}
		seasonID, _ = res.LastInsertId()
	} else if err != nil {
		return fmt.Errorf("find season: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		"INSERT INTO episodes (season_id, anime_id, title, number, url, type) VALUES (?, ?, ?, ?, ?, 'episode')",
		seasonID, animeID, ep.Title, ep.Number, ep.URL,
	)
	if err != nil {
		return fmt.Errorf("insert episode: %w", err)
	}

	epID, _ := res.LastInsertId()
	ep.ID = model.StringID(epID)
	ep.SeasonID = seasonID
	ep.AnimeID = animeID
	ep.SeasonNumber = seasonNumber

	return tx.Commit()
}

func (s *SQLiteAnimeStore) loadSeasons(ctx context.Context, animeID int64) ([]model.Season, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, anime_id, number, label, type FROM seasons WHERE anime_id = ? ORDER BY number",
		animeID,
	)
	if err != nil {
		return nil, fmt.Errorf("load seasons: %w", err)
	}
	defer rows.Close()

	seasons := []model.Season{}
	for rows.Next() {
		var se model.Season
		if err := rows.Scan(&se.ID, &se.AnimeID, &se.Number, &se.Label, &se.Type); err != nil {
			return nil, fmt.Errorf("scan season: %w", err)
		}

		episodes, err := s.loadEpisodes(ctx, se.ID, se.Number)
		if err != nil {
			return nil, err
		}
		se.Episodes = episodes
		seasons = append(seasons, se)
	}

	return seasons, nil
}

func genresToJSON(genres []string) string {
	if len(genres) == 0 {
		return "[]"
	}
	data, err := json.Marshal(genres)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func genresFromJSON(raw string) []string {
	var genres []string
	if err := json.Unmarshal([]byte(raw), &genres); err != nil {
		return nil
	}
	return genres
}

func (s *SQLiteAnimeStore) loadEpisodes(ctx context.Context, seasonID int64, seasonNumber int) ([]model.Episode, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, season_id, anime_id, title, number, url, type, storage_key, upscaled_storage_key FROM episodes WHERE season_id = ? ORDER BY CAST(number AS INTEGER), number",
		seasonID,
	)
	if err != nil {
		return nil, fmt.Errorf("load episodes: %w", err)
	}
	defer rows.Close()

	episodes := []model.Episode{}
	for rows.Next() {
		var ep model.Episode
		var id, epSeasonID, epAnimeID int64
		if err := rows.Scan(&id, &epSeasonID, &epAnimeID, &ep.Title, &ep.Number, &ep.URL, &ep.Type, &ep.StorageKey, &ep.UpscaledStorageKey); err != nil {
			return nil, fmt.Errorf("scan episode: %w", err)
		}
		ep.ID = model.StringID(id)
		ep.SeasonID = epSeasonID
		ep.AnimeID = epAnimeID
		ep.SeasonNumber = seasonNumber
		episodes = append(episodes, ep)
	}

	return episodes, nil
}
