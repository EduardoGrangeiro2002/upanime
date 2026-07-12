package scraper

import (
	"context"

	"upanime/api/model"
)

type Executor interface {
	Scrape(ctx context.Context, url string) (*model.Anime, error)
	Download(ctx context.Context, episodeURL string, destPath string, downloadID int64, dbPath string) error
}
