package scraper

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"upanime/api/model"
)

type PythonExecutor struct {
	scraperDir string
}

func NewPythonExecutor(scraperDir string) *PythonExecutor {
	return &PythonExecutor{scraperDir: scraperDir}
}

func (e *PythonExecutor) Scrape(ctx context.Context, url string) (*model.Anime, error) {
	cmd := exec.CommandContext(ctx, "uv", "run", "--directory", e.scraperDir, "python", "main.py", "scrape", url)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("scrape exec: %w", err)
	}

	return ParseScrapeOutput(out)
}

func (e *PythonExecutor) Download(ctx context.Context, episodeURL string, destPath string, downloadID int64, dbPath string) error {
	absDB, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}
	cmd := exec.CommandContext(ctx, "uv", "run", "--directory", e.scraperDir, "python", "main.py", "download",
		episodeURL, destPath, fmt.Sprintf("%d", downloadID), absDB,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("download exec: %w: %s", err, string(out))
	}

	return nil
}
