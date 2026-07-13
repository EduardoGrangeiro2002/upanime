package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"upanime/api/model"
	"upanime/api/scraper"
	"upanime/api/service"
	"upanime/api/store"
)

type AnimeHandler struct {
	scrapers  store.ScraperStore
	executor  scraper.Executor
	organizer *service.EpisodeOrganizer
}

func NewAnimeHandler(scrapers store.ScraperStore, executor scraper.Executor, organizer *service.EpisodeOrganizer) *AnimeHandler {
	return &AnimeHandler{scrapers: scrapers, executor: executor, organizer: organizer}
}

func (h *AnimeHandler) Get(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, `{"error":"url parameter required"}`, http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		http.Error(w, `{"error":"invalid url"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.scrapers.FindByDomain(r.Context(), parsed.Host); err != nil {
		http.Error(w, `{"error":"no scraper found for domain"}`, http.StatusNotFound)
		return
	}

	anime, err := h.executor.Scrape(r.Context(), rawURL)
	if err != nil {
		http.Error(w, `{"error":"scrape failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if h.organizer.Enabled() {
		if _, err := h.organizer.OrganizeAnime(r.Context(), anime); err != nil {
			log.Printf("episode organize failed for %s: %v", rawURL, err)
		}
	}

	ensureSeasons(anime)
	writeJSON(w, anime)
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
