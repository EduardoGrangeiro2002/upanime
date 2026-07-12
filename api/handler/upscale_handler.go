package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
)

type EditionHandler struct {
	jobs     store.UpscaleJobStore
	animes   store.AnimeStore
	episodes store.EpisodeStore
	storage  storage.FileStorage
	worker   service.UpscaleWorkerClient
}

var validTargetHeights = map[int]bool{1080: true, 1440: true, 2160: true}

func NewEditionHandler(
	jobs store.UpscaleJobStore,
	animes store.AnimeStore,
	episodes store.EpisodeStore,
	fs storage.FileStorage,
	worker service.UpscaleWorkerClient,
) *EditionHandler {
	return &EditionHandler{
		jobs:     jobs,
		animes:   animes,
		episodes: episodes,
		storage:  fs,
		worker:   worker,
	}
}

func (h *EditionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUpscaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.EpisodeIDs) == 0 {
		http.Error(w, `{"error":"episodeIds required"}`, http.StatusBadRequest)
		return
	}

	targetHeight, err := normalizeTargetHeight(req.TargetHeight)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	anime, err := h.animes.GetByID(r.Context(), req.AnimeID.Int64())
	if err != nil {
		http.Error(w, `{"error":"anime not found"}`, http.StatusNotFound)
		return
	}

	episodes := buildEpisodeMap(anime)
	created := make([]model.UpscaleJob, 0, len(req.EpisodeIDs))

	for _, episodeID := range req.EpisodeIDs {
		episode, ok := episodes[episodeID.Int64()]
		if !ok {
			http.Error(w, fmt.Sprintf(`{"error":"episode %d not found"}`, episodeID.Int64()), http.StatusBadRequest)
			return
		}

		if episode.StorageKey == "" {
			http.Error(w, fmt.Sprintf(`{"error":"episode %d not downloaded"}`, episodeID.Int64()), http.StatusBadRequest)
			return
		}

		job := &model.UpscaleJob{
			EpisodeID:        episodeID,
			AnimeID:          req.AnimeID,
			Type:             "upscale",
			TargetHeight:     targetHeight,
			BatchSize:        req.BatchSize,
			Sharpen:          req.Sharpen,
			Saturation:       req.Saturation,
			Contrast:         req.Contrast,
			Interpolate:      req.Interpolate,
			PanRatio:         req.PanRatio,
			Effects:          req.Effects,
			EffectsStrength:  req.EffectsStrength,
			EffectsSens:      req.EffectsSens,
			SourceStorageKey: episode.StorageKey,
			ResultStorageKey: buildUpscaledKey(episode.StorageKey),
			AnimeTitle:       anime.Title,
			AnimeImageURL:    anime.ImageURL,
			EpisodeTitle:     episode.Title,
			EpisodeNumber:    episode.Number,
			SeasonNumber:     episode.SeasonNumber,
		}

		if err := h.jobs.Create(r.Context(), job); err != nil {
			http.Error(w, `{"error":"create job failed"}`, http.StatusInternalServerError)
			return
		}

		created = append(created, *job)
		go h.dispatchJob(*job)
	}

	writeJSON(w, created)
}

func (h *EditionHandler) dispatchJob(job model.UpscaleJob) {
	ctx := context.Background()
	jobID := job.ID.Int64()

	sourceURL, err := h.storage.URL(ctx, job.SourceStorageKey)
	if err != nil {
		_ = h.jobs.UpdateStatus(ctx, jobID, "failed", fmt.Sprintf("Falha ao gerar URL fonte: %v", err))
		return
	}

	req := service.UpscaleWorkerJob{
		JobID:            jobID,
		SourceURL:        sourceURL,
		SourceStorageKey: job.SourceStorageKey,
		ResultStorageKey: job.ResultStorageKey,
		TargetHeight:     job.TargetHeight,
		BatchSize:        job.BatchSize,
		Sharpen:          job.Sharpen,
		Saturation:       job.Saturation,
		Contrast:         job.Contrast,
		Interpolate:      job.Interpolate,
		PanRatio:         job.PanRatio,
		Effects:          job.Effects,
		EffectsStrength:  job.EffectsStrength,
		EffectsSens:      job.EffectsSens,
	}

	runpodJobID, err := h.worker.Enqueue(ctx, req)
	if err != nil {
		_ = h.jobs.UpdateStatus(ctx, jobID, "failed", fmt.Sprintf("Falha ao enfileirar no RunPod: %v", err))
		return
	}

	_ = h.jobs.UpdateRunPodJobID(ctx, jobID, runpodJobID)
	_ = h.jobs.UpdateStatus(ctx, jobID, "processing", "")
}

func (h *EditionHandler) List(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobs.ListActive(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list jobs failed"}`, http.StatusInternalServerError)
		return
	}

	if jobs == nil {
		jobs = []model.UpscaleJob{}
	}

	writeJSON(w, jobs)
}

func (h *EditionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.jobs.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func buildEpisodeMap(anime *model.Anime) map[int64]model.Episode {
	result := make(map[int64]model.Episode)
	for _, season := range anime.Seasons {
		for _, episode := range season.Episodes {
			result[episode.ID.Int64()] = episode
		}
	}
	return result
}

func buildUpscaledKey(sourceKey string) string {
	ext := filepath.Ext(sourceKey)
	if ext == "" {
		return sourceKey + "_upscaled"
	}

	base := strings.TrimSuffix(sourceKey, ext)
	return base + "_upscaled" + ext
}

func normalizeTargetHeight(height int) (int, error) {
	if height == 0 {
		return 1080, nil
	}
	if !validTargetHeights[height] {
		return 0, fmt.Errorf("invalid targetHeight: must be 1080, 1440, or 2160")
	}
	return height, nil
}
