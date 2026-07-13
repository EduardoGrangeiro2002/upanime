package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/scraper"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
)

type DownloadHandler struct {
	downloads  store.DownloadStore
	animes     store.AnimeStore
	episodes   store.EpisodeStore
	scrapers   store.ScraperStore
	executor   scraper.Executor
	storage    storage.FileStorage
	classifier *service.GenreClassifier
	dbPath     string
	sem        chan struct{}
}

func NewDownloadHandler(
	downloads store.DownloadStore,
	animes store.AnimeStore,
	episodes store.EpisodeStore,
	scrapers store.ScraperStore,
	executor scraper.Executor,
	fs storage.FileStorage,
	classifier *service.GenreClassifier,
	dbPath string,
	concurrency int,
) *DownloadHandler {
	return &DownloadHandler{
		downloads:  downloads,
		animes:     animes,
		episodes:   episodes,
		scrapers:   scrapers,
		executor:   executor,
		storage:    fs,
		classifier: classifier,
		dbPath:     dbPath,
		sem:        make(chan struct{}, concurrency),
	}
}

func (h *DownloadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDownloadsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.Episodes) == 0 {
		http.Error(w, `{"error":"episodes required"}`, http.StatusBadRequest)
		return
	}

	anime, animeCreated, err := h.resolveAnime(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	episodes, err := h.resolveEpisodes(r.Context(), anime, req)
	if err != nil {
		http.Error(w, `{"error":"save episodes failed"}`, http.StatusInternalServerError)
		return
	}

	toCreate := make([]model.Download, len(episodes))
	for i, ep := range episodes {
		toCreate[i] = model.Download{EpisodeID: ep.ID, AnimeID: anime.ID}
	}

	created, err := h.downloads.Create(r.Context(), toCreate)
	if err != nil {
		http.Error(w, `{"error":"create downloads failed"}`, http.StatusInternalServerError)
		return
	}

	for i := range created {
		created[i].AnimeTitle = anime.Title
		created[i].AnimeImageURL = req.AnimeImageURL
		created[i].EpisodeTitle = episodes[i].Title
		created[i].EpisodeNumber = episodes[i].Number
		created[i].SeasonNumber = episodes[i].SeasonNumber
	}

	for _, d := range created {
		go h.processDownload(d, anime.Title)
	}

	if animeCreated && anime.ImageURL != "" {
		go h.downloadCoverAsync(anime.ID.Int64(), anime.ImageURL, anime.Title)
	}

	if animeCreated || len(anime.Genres) == 0 {
		h.classifier.ClassifyAsync(anime.ID.Int64())
	}

	writeJSON(w, created)
}

func (h *DownloadHandler) resolveAnime(ctx context.Context, req model.CreateDownloadsRequest) (*model.Anime, bool, error) {
	if req.AnimeID.Int64() > 0 {
		anime, err := h.animes.GetByID(ctx, req.AnimeID.Int64())
		if err != nil {
			return nil, false, errors.New("anime not found")
		}
		return anime, false, nil
	}

	title := strings.TrimSpace(req.AnimeTitle)
	if title == "" {
		return nil, false, errors.New("animeId or animeTitle required")
	}

	if existing, err := h.animes.FindByTitle(ctx, title); err == nil {
		return existing, false, nil
	}

	scraperID, err := h.scraperIDFor(ctx, req.SourceURL)
	if err != nil {
		return nil, false, err
	}

	anime := &model.Anime{
		Title:       title,
		URL:         req.SourceURL,
		ImageURL:    req.AnimeImageURL,
		Description: req.Description,
		ScraperID:   scraperID,
	}
	if err := h.animes.Create(ctx, anime); err != nil {
		anime.URL = "scrape://" + sanitize(title)
		if err := h.animes.Create(ctx, anime); err != nil {
			return nil, false, errors.New("create anime failed")
		}
	}
	return anime, true, nil
}

func (h *DownloadHandler) scraperIDFor(ctx context.Context, sourceURL string) (int64, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil || parsed.Host == "" {
		return 0, errors.New("valid sourceUrl required")
	}
	sc, err := h.scrapers.FindByDomain(ctx, parsed.Host)
	if err != nil {
		return 0, errors.New("no scraper found for domain")
	}
	return sc.ID, nil
}

func (h *DownloadHandler) resolveEpisodes(ctx context.Context, anime *model.Anime, req model.CreateDownloadsRequest) ([]model.Episode, error) {
	existing := make(map[string]model.Episode)
	for _, s := range anime.Seasons {
		for _, ep := range s.Episodes {
			existing[episodeKey(ep.SeasonNumber, ep.URL)] = ep
		}
	}

	var episodes []model.Episode
	for _, in := range req.Episodes {
		season := in.SeasonNumber
		if req.SeasonNumber > 0 {
			season = req.SeasonNumber
		}
		if season < 1 {
			season = 1
		}

		if ep, ok := existing[episodeKey(season, in.URL)]; ok {
			episodes = append(episodes, ep)
			continue
		}

		ep := model.Episode{Title: in.Title, Number: in.Number, URL: in.URL, Type: "episode"}
		if err := h.animes.AddEpisode(ctx, anime.ID.Int64(), season, &ep); err != nil {
			return nil, err
		}
		existing[episodeKey(season, ep.URL)] = ep
		episodes = append(episodes, ep)
	}
	return episodes, nil
}

func episodeKey(season int, url string) string {
	return fmt.Sprintf("%d|%s", season, url)
}

func (h *DownloadHandler) downloadCoverAsync(animeID int64, imageURL, title string) {
	ctx := context.Background()
	slug := sanitize(title)
	coverPath, err := service.DownloadCover(ctx, imageURL, slug, h.storage)
	if err != nil {
		log.Printf("cover download failed for anime %d: %v", animeID, err)
		return
	}
	if err := h.animes.UpdateCoverPath(ctx, animeID, coverPath); err != nil {
		log.Printf("update cover path failed for anime %d: %v", animeID, err)
	}
}

func (h *DownloadHandler) processDownload(d model.Download, animeTitle string) {
	h.sem <- struct{}{}
	defer func() { <-h.sem }()

	ctx := context.Background()

	_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "resolving", "")

	anime, err := h.animes.GetByID(ctx, d.AnimeID.Int64())
	if err != nil {
		_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "failed", err.Error())
		return
	}

	var episode *model.Episode
	for _, s := range anime.Seasons {
		for _, ep := range s.Episodes {
			if ep.ID.Int64() == d.EpisodeID.Int64() {
				episode = &ep
				break
			}
		}
	}

	if episode == nil {
		_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "failed", "episode not found")
		return
	}

	animeSlug := sanitize(animeTitle)
	epSlug := sanitize(episode.Title)
	if episode.Number != "" {
		epSlug = fmt.Sprintf("s%02de%s", episode.SeasonNumber, sanitize(episode.Number))
	}
	storageKey := fmt.Sprintf("animes/%s/%s.mp4", animeSlug, epSlug)
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("upanime_%d.mp4", d.ID.Int64()))

	err = h.executor.Download(ctx, episode.URL, tmpPath, d.ID.Int64(), h.dbPath)
	if err != nil {
		_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "failed", err.Error())
		return
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "failed", err.Error())
		return
	}
	defer f.Close()
	defer os.Remove(tmpPath)

	if err := h.storage.Save(ctx, storageKey, f); err != nil {
		_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "failed", err.Error())
		return
	}

	_ = h.episodes.UpdateStorageKey(ctx, d.EpisodeID.Int64(), storageKey)
	_ = h.downloads.UpdateStatus(ctx, d.ID.Int64(), "completed", "")
}

func (h *DownloadHandler) List(w http.ResponseWriter, r *http.Request) {
	downloads, err := h.downloads.ListActive(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list downloads failed"}`, http.StatusInternalServerError)
		return
	}
	if downloads == nil {
		downloads = []model.Download{}
	}
	writeJSON(w, downloads)
}

func (h *DownloadHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	d, err := h.downloads.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"download not found"}`, http.StatusNotFound)
		return
	}

	writeJSON(w, d)
}

func (h *DownloadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.downloads.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
