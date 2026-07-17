package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"upanime/api/model"
	"upanime/api/storage"
	"upanime/api/store"
)

var datasetClasses = map[string]bool{
	"fire":       true,
	"lightning":  true,
	"energy":     true,
	"aura":       true,
	"dark_magic": true,
	"beam":       true,
	"none":       true,
}

var datasetVerdicts = map[string]bool{
	"approved":   true,
	"rejected":   true,
	"needs_edit": true,
}

type DatasetHandler struct {
	samples store.DatasetStore
	storage storage.FileStorage
}

func NewDatasetHandler(samples store.DatasetStore, fs storage.FileStorage) *DatasetHandler {
	return &DatasetHandler{samples: samples, storage: fs}
}

func (h *DatasetHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, `{"error":"invalid multipart body"}`, http.StatusBadRequest)
		return
	}

	class := strings.TrimSpace(r.FormValue("class"))
	if !datasetClasses[class] {
		http.Error(w, `{"error":"invalid class"}`, http.StatusBadRequest)
		return
	}

	timestampS, err := parseOptionalFloat(r.FormValue("timestampS"))
	if err != nil {
		http.Error(w, `{"error":"invalid timestampS"}`, http.StatusBadRequest)
		return
	}
	teacherProb, err := parseOptionalFloat(r.FormValue("teacherProb"))
	if err != nil {
		http.Error(w, `{"error":"invalid teacherProb"}`, http.StatusBadRequest)
		return
	}

	frame, frameHeader, err := r.FormFile("frame")
	if err != nil {
		http.Error(w, `{"error":"frame file required"}`, http.StatusBadRequest)
		return
	}
	defer frame.Close()

	mask, _, err := r.FormFile("mask")
	if err != nil {
		http.Error(w, `{"error":"mask file required"}`, http.StatusBadRequest)
		return
	}
	defer mask.Close()

	sampleID, err := randomSampleID()
	if err != nil {
		http.Error(w, `{"error":"sample id generation failed"}`, http.StatusInternalServerError)
		return
	}

	frameExt := strings.ToLower(filepath.Ext(frameHeader.Filename))
	if frameExt != ".png" {
		frameExt = ".jpg"
	}
	frameKey := fmt.Sprintf("ml-dataset/frames/%s%s", sampleID, frameExt)
	maskKey := fmt.Sprintf("ml-dataset/masks/%s.png", sampleID)

	if err := h.storage.Save(r.Context(), frameKey, frame); err != nil {
		http.Error(w, `{"error":"frame upload failed"}`, http.StatusInternalServerError)
		return
	}
	if err := h.storage.Save(r.Context(), maskKey, mask); err != nil {
		http.Error(w, `{"error":"mask upload failed"}`, http.StatusInternalServerError)
		return
	}

	sample := &model.DatasetSample{
		Source:      strings.TrimSpace(r.FormValue("source")),
		Class:       class,
		FrameKey:    frameKey,
		MaskKey:     maskKey,
		AnimeTitle:  strings.TrimSpace(r.FormValue("animeTitle")),
		Episode:     strings.TrimSpace(r.FormValue("episode")),
		TimestampS:  timestampS,
		TeacherProb: teacherProb,
	}
	if err := h.samples.Create(r.Context(), sample); err != nil {
		http.Error(w, `{"error":"create sample failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sample)
}

func (h *DatasetHandler) Queue(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			http.Error(w, `{"error":"invalid limit"}`, http.StatusBadRequest)
			return
		}
		limit = min(n, 100)
	}

	samples, err := h.samples.Queue(r.Context(), limit)
	if err != nil {
		http.Error(w, `{"error":"list queue failed"}`, http.StatusInternalServerError)
		return
	}

	for i := range samples {
		frameURL, err := h.storage.URL(r.Context(), samples[i].FrameKey)
		if err != nil {
			http.Error(w, `{"error":"frame url failed"}`, http.StatusInternalServerError)
			return
		}
		maskURL, err := h.storage.URL(r.Context(), samples[i].MaskKey)
		if err != nil {
			http.Error(w, `{"error":"mask url failed"}`, http.StatusInternalServerError)
			return
		}
		samples[i].FrameURL = frameURL
		samples[i].MaskURL = maskURL
	}

	writeJSON(w, samples)
}

func (h *DatasetHandler) Verdict(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		Verdict string `json:"verdict"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if !datasetVerdicts[body.Verdict] {
		http.Error(w, `{"error":"invalid verdict"}`, http.StatusBadRequest)
		return
	}

	if err := h.samples.SetVerdict(r.Context(), id, body.Verdict); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"sample not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"verdict failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DatasetHandler) Stats(w http.ResponseWriter, r *http.Request) {
	rows, err := h.samples.Stats(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats failed"}`, http.StatusInternalServerError)
		return
	}

	stats := model.DatasetStats{ByClass: rows}
	for _, row := range rows {
		stats.Total += row.Count
		switch row.Status {
		case "pending":
			stats.Pending += row.Count
		case "approved":
			stats.Approved += row.Count
		case "rejected":
			stats.Rejected += row.Count
		case "needs_edit":
			stats.NeedsEdit += row.Count
		}
	}

	writeJSON(w, stats)
}

func parseOptionalFloat(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func randomSampleID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
