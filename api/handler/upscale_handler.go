package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hibiken/asynq"
	"upanime/api/jobs"
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
	enq      jobs.Enqueuer
}

var validTargetHeights = map[int]bool{1080: true, 1440: true, 2160: true}

func NewEditionHandler(
	jobStore store.UpscaleJobStore,
	animes store.AnimeStore,
	episodes store.EpisodeStore,
	fs storage.FileStorage,
	worker service.UpscaleWorkerClient,
	enq jobs.Enqueuer,
) *EditionHandler {
	return &EditionHandler{
		jobs:     jobStore,
		animes:   animes,
		episodes: episodes,
		storage:  fs,
		worker:   worker,
		enq:      enq,
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

	if req.Upscaler != "" && req.Upscaler != "compact" && req.Upscaler != "apisr" {
		http.Error(w, `{"error":"upscaler must be compact or apisr"}`, http.StatusBadRequest)
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
			SkipUpscale:      req.SkipUpscale,
			Upscaler:         req.Upscaler,
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
		if err := h.enq.EnqueueUpscaleDispatch(r.Context(), buildWorkerJob(*job)); err != nil {
			_ = h.jobs.UpdateStatus(r.Context(), job.ID.Int64(), "failed", "enfileirar: "+err.Error())
		}
	}

	writeJSON(w, created)
}

func buildWorkerJob(job model.UpscaleJob) service.UpscaleWorkerJob {
	return service.UpscaleWorkerJob{
		JobID:            job.ID.Int64(),
		SourceStorageKey: job.SourceStorageKey,
		ResultStorageKey: job.ResultStorageKey,
		Variants:         buildWorkerVariants(job),
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
		SkipUpscale:      job.SkipUpscale,
		Upscaler:         job.Upscaler,
	}
}

func (h *EditionHandler) ProcessDispatchTask(ctx context.Context, t *asynq.Task) error {
	var wj service.UpscaleWorkerJob
	if err := json.Unmarshal(t.Payload(), &wj); err != nil {
		return err
	}

	job, err := h.jobs.GetByID(ctx, wj.JobID)
	if err != nil {
		return nil
	}
	if job.RunPodJobID != "" {
		return nil
	}

	sourceURL, err := h.storage.URL(ctx, wj.SourceStorageKey)
	if err != nil {
		return h.dispatchFailed(ctx, wj.JobID, fmt.Sprintf("Falha ao gerar URL fonte: %v", err))
	}
	wj.SourceURL = sourceURL

	runpodJobID, err := h.worker.Enqueue(ctx, wj)
	if err != nil {
		return h.dispatchFailed(ctx, wj.JobID, fmt.Sprintf("Falha ao enfileirar no RunPod: %v", err))
	}

	_ = h.jobs.UpdateRunPodJobID(ctx, wj.JobID, runpodJobID)
	_ = h.jobs.UpdateStatus(ctx, wj.JobID, "processing", "")
	return nil
}

func (h *EditionHandler) dispatchFailed(ctx context.Context, jobID int64, msg string) error {
	if jobs.FinalAttempt(ctx) {
		_ = h.jobs.UpdateStatus(ctx, jobID, "failed", msg)
		return errors.New(msg)
	}
	_ = h.jobs.UpdateStatus(ctx, jobID, "queued", msg)
	return errors.New(msg)
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

	_ = h.enq.CancelUpscale(r.Context(), id)

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

func buildWorkerVariants(job model.UpscaleJob) []service.WorkerVariant {
	if job.SkipUpscale {
		return nil
	}
	heights := model.VariantHeights(job.TargetHeight)
	variants := make([]service.WorkerVariant, 0, len(heights))
	for _, h := range heights {
		variants = append(variants, service.WorkerVariant{
			Height:     h,
			StorageKey: model.BuildVariantKey(job.ResultStorageKey, h),
		})
	}
	return variants
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
