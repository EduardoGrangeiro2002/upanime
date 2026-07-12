package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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

	if len(req.EpisodeIDs) == 0 {
		http.Error(w, `{"error":"episodeIds required"}`, http.StatusBadRequest)
		return
	}

	var toCreate []model.Download
	for _, epID := range req.EpisodeIDs {
		toCreate = append(toCreate, model.Download{
			EpisodeID: epID,
			AnimeID:   req.AnimeID,
		})
	}

	created, err := h.downloads.Create(r.Context(), toCreate)
	if err != nil {
		http.Error(w, `{"error":"create downloads failed"}`, http.StatusInternalServerError)
		return
	}

	anime, _ := h.animes.GetByID(r.Context(), req.AnimeID.Int64())
	episodeMap := make(map[int64]model.Episode)
	if anime != nil {
		for _, s := range anime.Seasons {
			for _, ep := range s.Episodes {
				episodeMap[ep.ID.Int64()] = ep
			}
		}
	}

	for i := range created {
		created[i].AnimeTitle = req.AnimeTitle
		created[i].AnimeImageURL = req.AnimeImageURL
		if ep, ok := episodeMap[created[i].EpisodeID.Int64()]; ok {
			created[i].EpisodeTitle = ep.Title
			created[i].EpisodeNumber = ep.Number
			created[i].SeasonNumber = ep.SeasonNumber
		}
	}

	for _, d := range created {
		go h.processDownload(d, req.AnimeTitle)
	}

	h.classifier.ClassifyAsync(req.AnimeID.Int64())

	writeJSON(w, created)
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
