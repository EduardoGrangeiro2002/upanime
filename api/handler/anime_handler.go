package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"upanime/api/model"
	"upanime/api/scraper"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
)

type AnimeHandler struct {
	animes   store.AnimeStore
	scrapers store.ScraperStore
	executor scraper.Executor
	storage  storage.FileStorage
}

func NewAnimeHandler(animes store.AnimeStore, scrapers store.ScraperStore, executor scraper.Executor, fs storage.FileStorage) *AnimeHandler {
	return &AnimeHandler{animes: animes, scrapers: scrapers, executor: executor, storage: fs}
}

func (h *AnimeHandler) Get(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, `{"error":"url parameter required"}`, http.StatusBadRequest)
		return
	}

	existing, err := h.animes.FindByURL(r.Context(), rawURL)
	if err == nil {
		h.populateCoverURL(r.Context(), existing)
		ensureSeasons(existing)
		writeJSON(w, existing)
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		http.Error(w, `{"error":"invalid url"}`, http.StatusBadRequest)
		return
	}

	sc, err := h.scrapers.FindByDomain(r.Context(), parsed.Host)
	if err != nil {
		http.Error(w, `{"error":"no scraper found for domain"}`, http.StatusNotFound)
		return
	}

	anime, err := h.executor.Scrape(r.Context(), rawURL)
	if err != nil {
		http.Error(w, `{"error":"scrape failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	anime.ScraperID = sc.ID
	if err := h.animes.Create(r.Context(), anime); err != nil {
		http.Error(w, `{"error":"save failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if anime.ImageURL != "" {
		go h.downloadCoverAsync(anime.ID.Int64(), anime.ImageURL, anime.Title)
	}

	saved, err := h.animes.GetByID(r.Context(), anime.ID.Int64())
	if err != nil {
		ensureSeasons(anime)
		writeJSON(w, anime)
		return
	}

	ensureSeasons(saved)
	writeJSON(w, saved)
}

func (h *AnimeHandler) downloadCoverAsync(animeID int64, imageURL, title string) {
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

func (h *AnimeHandler) populateCoverURL(ctx context.Context, a *model.Anime) {
	if a.CoverPath == "" {
		return
	}
	url, err := h.storage.URL(ctx, a.CoverPath)
	if err != nil {
		return
	}
	a.CoverURL = url
}

func ensureSeasons(a *model.Anime) {
	if a.Seasons == nil {
		a.Seasons = []model.Season{}
	}
	for i := range a.Seasons {
		if a.Seasons[i].Episodes == nil {
			a.Seasons[i].Episodes = []model.Episode{}
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func sanitize(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return r.Replace(s)
}
