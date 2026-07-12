package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
)

type UploadHandler struct {
	animes     store.AnimeStore
	episodes   store.EpisodeStore
	scrapers   store.ScraperStore
	storage    storage.FileStorage
	classifier *service.GenreClassifier
}

func NewUploadHandler(
	animes store.AnimeStore,
	episodes store.EpisodeStore,
	scrapers store.ScraperStore,
	fs storage.FileStorage,
	classifier *service.GenreClassifier,
) *UploadHandler {
	return &UploadHandler{
		animes:     animes,
		episodes:   episodes,
		scrapers:   scrapers,
		storage:    fs,
		classifier: classifier,
	}
}

type uploadResponse struct {
	AnimeID   model.StringID `json:"animeId"`
	Episode   model.Episode  `json:"episode"`
	Replaced  bool           `json:"replaced"`
}

func (h *UploadHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, `{"error":"invalid multipart body"}`, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("animeTitle"))
	if title == "" {
		http.Error(w, `{"error":"animeTitle required"}`, http.StatusBadRequest)
		return
	}

	episodeNumber := strings.TrimSpace(r.FormValue("episodeNumber"))
	if episodeNumber == "" {
		http.Error(w, `{"error":"episodeNumber required"}`, http.StatusBadRequest)
		return
	}

	seasonNumber := 1
	if raw := strings.TrimSpace(r.FormValue("seasonNumber")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			http.Error(w, `{"error":"invalid seasonNumber"}`, http.StatusBadRequest)
			return
		}
		seasonNumber = n
	}

	episodeTitle := strings.TrimSpace(r.FormValue("episodeTitle"))
	if episodeTitle == "" {
		episodeTitle = fmt.Sprintf("Episódio %s", episodeNumber)
	}

	description := strings.TrimSpace(r.FormValue("description"))

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := videoExtension(header.Filename)
	if ext == "" {
		http.Error(w, `{"error":"unsupported video format, use mp4, webm ou mkv"}`, http.StatusBadRequest)
		return
	}

	anime, created, err := h.ensureAnime(r, title, description)
	if err != nil {
		http.Error(w, `{"error":"save anime failed"}`, http.StatusInternalServerError)
		return
	}

	episode, replaced := findEpisode(anime, seasonNumber, episodeNumber)
	if episode == nil {
		episode = &model.Episode{
			Title:  episodeTitle,
			Number: episodeNumber,
			URL:    fmt.Sprintf("upload://%s/s%de%s", sanitize(title), seasonNumber, episodeNumber),
			Type:   "episode",
		}
		if err := h.animes.AddEpisode(r.Context(), anime.ID.Int64(), seasonNumber, episode); err != nil {
			http.Error(w, `{"error":"save episode failed"}`, http.StatusInternalServerError)
			return
		}
	}

	storageKey := fmt.Sprintf("animes/%s/s%02de%s%s", sanitize(title), seasonNumber, sanitize(episodeNumber), ext)
	if err := h.storage.Save(r.Context(), storageKey, file); err != nil {
		http.Error(w, `{"error":"save file failed"}`, http.StatusInternalServerError)
		return
	}

	if replaced && episode.StorageKey != "" && episode.StorageKey != storageKey {
		_ = h.storage.Delete(r.Context(), episode.StorageKey)
	}

	if err := h.episodes.UpdateStorageKey(r.Context(), episode.ID.Int64(), storageKey); err != nil {
		http.Error(w, `{"error":"update episode failed"}`, http.StatusInternalServerError)
		return
	}
	episode.StorageKey = storageKey

	if created || len(anime.Genres) == 0 {
		h.classifier.ClassifyAsync(anime.ID.Int64())
	}

	writeJSON(w, uploadResponse{AnimeID: anime.ID, Episode: *episode, Replaced: replaced})
}

func (h *UploadHandler) ensureAnime(r *http.Request, title, description string) (*model.Anime, bool, error) {
	syntheticURL := "upload://" + sanitize(title)

	existing, err := h.animes.FindByURL(r.Context(), syntheticURL)
	if err == nil {
		return existing, false, nil
	}

	sc, err := h.scrapers.FindByDomain(r.Context(), "upload")
	if err != nil {
		return nil, false, fmt.Errorf("find upload scraper: %w", err)
	}

	anime := &model.Anime{
		Title:       title,
		URL:         syntheticURL,
		Description: description,
		ScraperID:   sc.ID,
	}
	if err := h.animes.Create(r.Context(), anime); err != nil {
		return nil, false, fmt.Errorf("create anime: %w", err)
	}
	return anime, true, nil
}

func findEpisode(anime *model.Anime, seasonNumber int, episodeNumber string) (*model.Episode, bool) {
	for _, s := range anime.Seasons {
		if s.Number != seasonNumber || s.Type != "episode" {
			continue
		}
		for i := range s.Episodes {
			if s.Episodes[i].Number == episodeNumber {
				return &s.Episodes[i], true
			}
		}
	}
	return nil, false
}

func videoExtension(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return ""
	}
	ext := strings.ToLower(name[idx:])
	switch ext {
	case ".mp4", ".webm", ".mkv", ".m4v", ".mov":
		return ext
	}
	return ""
}
