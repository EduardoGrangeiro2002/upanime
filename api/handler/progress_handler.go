package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/store"
)

const progressListLimit = 20

type ProgressHandler struct {
	progress store.WatchProgressStore
}

func NewProgressHandler(progress store.WatchProgressStore) *ProgressHandler {
	return &ProgressHandler{progress: progress}
}

type progressUpdateRequest struct {
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
}

func episodeIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func (h *ProgressHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := episodeIDParam(r)
	if err != nil {
		writeJSONError(w, "id de episódio inválido", http.StatusBadRequest)
		return
	}

	var req progressUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	if req.Position < 0 || req.Duration < 0 {
		writeJSONError(w, "posição inválida", http.StatusBadRequest)
		return
	}

	err = h.progress.Upsert(r.Context(), UserEmail(r.Context()), id, req.Position, req.Duration)
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if strings.Contains(err.Error(), "FOREIGN KEY") {
		writeJSONError(w, "episódio não encontrado", http.StatusNotFound)
		return
	}
	writeJSONError(w, "erro ao salvar progresso", http.StatusInternalServerError)
}

func (h *ProgressHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := episodeIDParam(r)
	if err != nil {
		writeJSONError(w, "id de episódio inválido", http.StatusBadRequest)
		return
	}

	p, err := h.progress.Get(r.Context(), UserEmail(r.Context()), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "progresso não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		writeJSONError(w, "erro ao buscar progresso", http.StatusInternalServerError)
		return
	}

	writeJSON(w, p)
}

func (h *ProgressHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.progress.ListInProgress(r.Context(), UserEmail(r.Context()), progressListLimit)
	if err != nil {
		writeJSONError(w, "erro ao listar progresso", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []model.WatchProgress{}
	}

	writeJSON(w, items)
}
