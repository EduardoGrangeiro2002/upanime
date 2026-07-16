package store

import (
	"context"
	"time"

	"upanime/api/model"
)

type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context) ([]model.User, error)
	UpdatePassword(ctx context.Context, email, passwordHash string, mustChange bool) error
	UpdateMFAContext(ctx context.Context, email, ip, location string, at time.Time) error
}

type AnimeStore interface {
	FindByURL(ctx context.Context, url string) (*model.Anime, error)
	FindByTitle(ctx context.Context, title string) (*model.Anime, error)
	Create(ctx context.Context, anime *model.Anime) error
	GetByID(ctx context.Context, id int64) (*model.Anime, error)
	List(ctx context.Context) ([]model.Anime, error)
	Delete(ctx context.Context, id int64) error
	UpdateCoverPath(ctx context.Context, id int64, path string) error
	UpdateGenres(ctx context.Context, id int64, genres []string) error
	AddEpisode(ctx context.Context, animeID int64, seasonNumber int, ep *model.Episode) error
	UpdateEpisodeNumber(ctx context.Context, episodeID int64, number string) error
}

type DownloadStore interface {
	Create(ctx context.Context, downloads []model.Download) ([]model.Download, error)
	ListActive(ctx context.Context) ([]model.Download, error)
	GetByID(ctx context.Context, id int64) (*model.Download, error)
	UpdateStatus(ctx context.Context, id int64, status string, errMsg string) error
	Delete(ctx context.Context, id int64) error
}

type EpisodeStore interface {
	GetByID(ctx context.Context, id int64) (*model.Episode, error)
	Delete(ctx context.Context, id int64) error
	UpdateStorageKey(ctx context.Context, id int64, key string) error
	UpdateUpscaledStorageKey(ctx context.Context, id int64, key string) error
	UpdateUpscaledVariants(ctx context.Context, id int64, variants []model.EpisodeVariant) error
}

type ScraperStore interface {
	FindByDomain(ctx context.Context, domain string) (*model.Scraper, error)
}

type UpscaleJobStore interface {
	Create(ctx context.Context, job *model.UpscaleJob) error
	ListActive(ctx context.Context) ([]model.UpscaleJob, error)
	ListProcessing(ctx context.Context) ([]model.UpscaleJob, error)
	GetByID(ctx context.Context, id int64) (*model.UpscaleJob, error)
	UpdateStatus(ctx context.Context, id int64, status string, errMsg string) error
	UpdateResult(ctx context.Context, id int64, resultKey string) error
	UpdateRunPodJobID(ctx context.Context, id int64, runpodJobID string) error
	Delete(ctx context.Context, id int64) error
}
