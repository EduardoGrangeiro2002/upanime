package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hibiken/asynq"
	"upanime/api/jobs"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/store"
)

func OrganizeAnimeHandler(organizer *service.EpisodeOrganizer, animes store.AnimeStore, enq jobs.Enqueuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !organizer.Enabled() {
			http.Error(w, `{"error":"organizador desativado: OPENROUTER_API_KEY não configurada"}`, http.StatusServiceUnavailable)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}

		if _, err := animes.GetByID(r.Context(), id); err != nil {
			http.Error(w, `{"error":"anime not found"}`, http.StatusNotFound)
			return
		}

		if err := enq.EnqueueOrganize(r.Context(), id); err != nil {
			http.Error(w, `{"error":"enfileirar organização falhou"}`, http.StatusInternalServerError)
			return
		}

		writeAccepted(w)
	}
}

func OrganizeTask(organizer *service.EpisodeOrganizer, animes store.AnimeStore) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p jobs.OrganizePayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		return runOrganize(ctx, organizer, animes, p.AnimeID)
	}
}

func runOrganize(ctx context.Context, organizer *service.EpisodeOrganizer, animes store.AnimeStore, animeID int64) error {
	anime, err := animes.GetByID(ctx, animeID)
	if err != nil {
		return nil
	}

	before := snapshotNumbers(anime)
	if _, err := organizer.OrganizeAnime(ctx, anime); err != nil {
		return err
	}

	for _, ep := range changedEpisodes(anime, before) {
		if err := animes.UpdateEpisodeNumber(ctx, ep.ID.Int64(), ep.Number); err != nil {
			return err
		}
	}
	return nil
}

func snapshotNumbers(anime *model.Anime) map[int64]string {
	numbers := make(map[int64]string)
	for _, season := range anime.Seasons {
		for _, ep := range season.Episodes {
			numbers[ep.ID.Int64()] = ep.Number
		}
	}
	return numbers
}

func changedEpisodes(anime *model.Anime, before map[int64]string) []model.Episode {
	var changed []model.Episode
	for _, season := range anime.Seasons {
		for _, ep := range season.Episodes {
			if before[ep.ID.Int64()] == ep.Number {
				continue
			}
			changed = append(changed, ep)
		}
	}
	return changed
}
