package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
)

type ThumbnailHandler struct {
	episodes store.EpisodeStore
	thumbs   *service.ThumbnailService
	storage  storage.FileStorage
}

func NewThumbnailHandler(episodes store.EpisodeStore, thumbs *service.ThumbnailService, fs storage.FileStorage) *ThumbnailHandler {
	return &ThumbnailHandler{episodes: episodes, thumbs: thumbs, storage: fs}
}

func (h *ThumbnailHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	episode, err := h.episodes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"episode not found"}`, http.StatusNotFound)
		return
	}
	if episode.StorageKey == "" {
		http.Error(w, `{"error":"episode not downloaded"}`, http.StatusNotFound)
		return
	}

	thumbKey, err := h.thumbs.Ensure(r.Context(), episode.StorageKey)
	if err != nil {
		log.Printf("thumbnail %s: %v", episode.StorageKey, err)
		http.Error(w, `{"error":"thumbnail generation failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	if err := h.storage.ServeFile(r.Context(), w, r, thumbKey); err != nil {
		log.Printf("serve thumbnail %s: %v", thumbKey, err)
	}
}
