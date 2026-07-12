package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/store"
)

func OrganizeAnimeHandler(organizer *service.EpisodeOrganizer, animes store.AnimeStore) http.HandlerFunc {
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

		anime, err := animes.GetByID(r.Context(), id)
		if err != nil {
			http.Error(w, `{"error":"anime not found"}`, http.StatusNotFound)
			return
		}

		before := snapshotNumbers(anime)
		if _, err := organizer.OrganizeAnime(r.Context(), anime); err != nil {
			http.Error(w, `{"error":"organize failed: `+err.Error()+`"}`, http.StatusBadGateway)
			return
		}

		for _, ep := range changedEpisodes(anime, before) {
			if err := animes.UpdateEpisodeNumber(r.Context(), ep.ID.Int64(), ep.Number); err != nil {
				http.Error(w, `{"error":"save failed: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
		}

		updated, err := animes.GetByID(r.Context(), id)
		if err != nil {
			http.Error(w, `{"error":"reload failed: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		ensureSeasons(updated)
		writeJSON(w, updated)
	}
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
