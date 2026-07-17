package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/db"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/storage"
	"upanime/api/store"
)

func setupDatasetTest(t *testing.T) (*handler.DatasetHandler, *chi.Mux) {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "ml_dataset.db"))
	if err != nil {
		t.Fatalf("open dataset db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	datasetStore, err := store.NewSQLiteDatasetStore(database)
	if err != nil {
		t.Fatalf("create dataset store: %v", err)
	}

	datasetHandler := handler.NewDatasetHandler(datasetStore, storage.NewLocalStorage(t.TempDir()))

	router := chi.NewRouter()
	router.Post("/api/dataset/samples", datasetHandler.Ingest)
	router.Get("/api/dataset/samples/queue", datasetHandler.Queue)
	router.Post("/api/dataset/samples/{id}/verdict", datasetHandler.Verdict)
	router.Get("/api/dataset/stats", datasetHandler.Stats)
	return datasetHandler, router
}

func ingestSampleRequest(t *testing.T, class string) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	form.WriteField("class", class)
	form.WriteField("animeTitle", "Slayers")
	form.WriteField("episode", "S1E04")
	form.WriteField("timestampS", "54.3")
	form.WriteField("teacherProb", "0.42")

	frame, _ := form.CreateFormFile("frame", "frame.jpg")
	io.Copy(frame, strings.NewReader("fake-jpg-bytes"))
	mask, _ := form.CreateFormFile("mask", "mask.png")
	io.Copy(mask, strings.NewReader("fake-png-bytes"))
	form.Close()

	request := httptest.NewRequest("POST", "/api/dataset/samples", body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}

func TestDatasetHandler_IngestQueueVerdictStats(t *testing.T) {
	_, router := setupDatasetTest(t)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, ingestSampleRequest(t, "fire"))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}

	var created model.DatasetSample
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode created sample: %v", err)
	}
	if created.Status != "pending" || created.Class != "fire" {
		t.Errorf("unexpected sample: %+v", created)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/dataset/samples/queue", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var queue []model.DatasetSample
	if err := json.NewDecoder(response.Body).Decode(&queue); err != nil {
		t.Fatalf("decode queue: %v", err)
	}
	if len(queue) != 1 {
		t.Fatalf("expected 1 sample in queue, got %d", len(queue))
	}
	if queue[0].FrameURL == "" || queue[0].MaskURL == "" {
		t.Errorf("expected frame and mask urls, got %+v", queue[0])
	}
	if queue[0].AnimeTitle != "Slayers" || queue[0].TimestampS != 54.3 || queue[0].TeacherProb != 0.42 {
		t.Errorf("metadata not preserved: %+v", queue[0])
	}

	verdictBody := strings.NewReader(`{"verdict":"approved"}`)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", fmt.Sprintf("/api/dataset/samples/%d/verdict", created.ID.Int64()), verdictBody))
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/dataset/samples/queue", nil))
	json.NewDecoder(response.Body).Decode(&queue)
	if len(queue) != 0 {
		t.Errorf("expected empty queue after verdict, got %d", len(queue))
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/dataset/stats", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var stats model.DatasetStats
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Total != 1 || stats.Approved != 1 || stats.Pending != 0 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}

func TestDatasetHandler_IngestRejectsInvalidClass(t *testing.T) {
	_, router := setupDatasetTest(t)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, ingestSampleRequest(t, "hair"))

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", response.Code)
	}
}

func TestDatasetHandler_IngestRequiresMask(t *testing.T) {
	_, router := setupDatasetTest(t)

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	form.WriteField("class", "fire")
	frame, _ := form.CreateFormFile("frame", "frame.jpg")
	io.Copy(frame, strings.NewReader("fake"))
	form.Close()

	request := httptest.NewRequest("POST", "/api/dataset/samples", body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", response.Code)
	}
}

func TestDatasetHandler_VerdictValidation(t *testing.T) {
	_, router := setupDatasetTest(t)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/dataset/samples/1/verdict", strings.NewReader(`{"verdict":"maybe"}`)))
	if response.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid verdict, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/dataset/samples/999/verdict", strings.NewReader(`{"verdict":"rejected"}`)))
	if response.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown sample, got %d", response.Code)
	}
}
