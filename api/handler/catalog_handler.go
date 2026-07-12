package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/storage"
	"upanime/api/store"
)

type CatalogHandler struct {
	animes   store.AnimeStore
	episodes store.EpisodeStore
	storage  storage.FileStorage
}

func NewCatalogHandler(animes store.AnimeStore, episodes store.EpisodeStore, fs storage.FileStorage) *CatalogHandler {
	return &CatalogHandler{animes: animes, episodes: episodes, storage: fs}
}

func (h *CatalogHandler) List(w http.ResponseWriter, r *http.Request) {
	animes, err := h.animes.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list animes failed"}`, http.StatusInternalServerError)
		return
	}

	var result []model.Anime
	for _, a := range animes {
		if !hasDownloadedEpisodes(&a) {
			continue
		}

		if a.CoverPath != "" {
			url, err := h.storage.URL(r.Context(), a.CoverPath)
			if err == nil {
				a.CoverURL = url
			}
		}

		result = append(result, a)
	}

	if result == nil {
		result = []model.Anime{}
	}
	writeJSON(w, result)
}

func (h *CatalogHandler) DeleteAnime(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	anime, err := h.animes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"anime not found"}`, http.StatusNotFound)
		return
	}

	h.deleteAnimeFiles(r.Context(), anime)

	if err := h.animes.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete anime failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CatalogHandler) DeleteEpisode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	ep, err := h.episodes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"episode not found"}`, http.StatusNotFound)
		return
	}

	if ep.StorageKey != "" {
		if err := h.storage.Delete(r.Context(), ep.StorageKey); err != nil {
			log.Printf("delete episode file %s: %v", ep.StorageKey, err)
		}
	}

	if ep.UpscaledStorageKey != "" {
		if err := h.storage.Delete(r.Context(), ep.UpscaledStorageKey); err != nil {
			log.Printf("delete upscaled episode file %s: %v", ep.UpscaledStorageKey, err)
		}
	}

	if err := h.episodes.UpdateStorageKey(r.Context(), id, ""); err != nil {
		http.Error(w, `{"error":"clear storage key failed"}`, http.StatusInternalServerError)
		return
	}

	if err := h.episodes.UpdateUpscaledStorageKey(r.Context(), id, ""); err != nil {
		http.Error(w, `{"error":"clear upscaled storage key failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CatalogHandler) DeleteUpscaledEpisode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	ep, err := h.episodes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"episode not found"}`, http.StatusNotFound)
		return
	}

	if ep.UpscaledStorageKey == "" {
		http.Error(w, `{"error":"episode not upscaled"}`, http.StatusNotFound)
		return
	}

	if err := h.storage.Delete(r.Context(), ep.UpscaledStorageKey); err != nil {
		log.Printf("delete upscaled episode file %s: %v", ep.UpscaledStorageKey, err)
	}

	if err := h.episodes.UpdateUpscaledStorageKey(r.Context(), id, ""); err != nil {
		http.Error(w, `{"error":"clear upscaled storage key failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CatalogHandler) deleteAnimeFiles(ctx context.Context, anime *model.Anime) {
	if anime.CoverPath != "" {
		if err := h.storage.Delete(ctx, anime.CoverPath); err != nil {
			log.Printf("delete cover %s: %v", anime.CoverPath, err)
		}
	}

	for _, s := range anime.Seasons {
		for _, ep := range s.Episodes {
			if ep.StorageKey != "" {
				if err := h.storage.Delete(ctx, ep.StorageKey); err != nil {
					log.Printf("delete episode file %s: %v", ep.StorageKey, err)
				}
			}

			if ep.UpscaledStorageKey == "" {
				continue
			}

			if err := h.storage.Delete(ctx, ep.UpscaledStorageKey); err != nil {
				log.Printf("delete upscaled episode file %s: %v", ep.UpscaledStorageKey, err)
			}
		}
	}
}

func (h *CatalogHandler) UploadCover(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	anime, err := h.animes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"anime not found"}`, http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, `{"error":"file too large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("cover")
	if err != nil {
		http.Error(w, `{"error":"cover file required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := extensionFromFilename(header.Filename)
	slug := sanitize(anime.Title)
	key := "animes/" + slug + "/cover" + ext

	if anime.CoverPath != "" && anime.CoverPath != key {
		_ = h.storage.Delete(r.Context(), anime.CoverPath)
	}

	if err := h.storage.Save(r.Context(), key, file); err != nil {
		http.Error(w, `{"error":"save cover failed"}`, http.StatusInternalServerError)
		return
	}

	if err := h.animes.UpdateCoverPath(r.Context(), id, key); err != nil {
		http.Error(w, `{"error":"update cover path failed"}`, http.StatusInternalServerError)
		return
	}

	coverURL, err := h.storage.URL(r.Context(), key)
	if err != nil {
		http.Error(w, `{"error":"generate url failed"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"coverUrl": coverURL})
}

func (h *CatalogHandler) StreamURL(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	ep, err := h.episodes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"episode not found"}`, http.StatusNotFound)
		return
	}

	key := episodeStorageKeyForVariant(ep, r.URL.Query().Get("variant"))
	if key == "" {
		http.Error(w, `{"error":"episode stream not available"}`, http.StatusNotFound)
		return
	}

	url, err := h.storage.URL(r.Context(), key)
	if err != nil {
		http.Error(w, `{"error":"generate url failed"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"url": url})
}

func (h *CatalogHandler) StreamFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	ep, err := h.episodes.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"episode not found"}`, http.StatusNotFound)
		return
	}

	key := episodeStorageKeyForVariant(ep, r.URL.Query().Get("variant"))
	if key == "" {
		http.Error(w, `{"error":"episode stream not available"}`, http.StatusNotFound)
		return
	}

	if err := h.storage.ServeFile(r.Context(), w, r, key); err != nil {
		log.Printf("stream file %s: %v", key, err)
		http.Error(w, `{"error":"stream failed"}`, http.StatusInternalServerError)
	}
}

func extensionFromFilename(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			ext := name[i:]
			switch ext {
			case ".jpg", ".jpeg", ".png", ".webp":
				return ext
			}
			break
		}
	}
	return ".jpg"
}

func hasDownloadedEpisodes(a *model.Anime) bool {
	for _, s := range a.Seasons {
		for _, ep := range s.Episodes {
			if ep.StorageKey != "" {
				return true
			}
		}
	}
	return false
}

func episodeStorageKeyForVariant(ep *model.Episode, variant string) string {
	if variant == "upscaled" {
		return ep.UpscaledStorageKey
	}
	return ep.StorageKey
}
