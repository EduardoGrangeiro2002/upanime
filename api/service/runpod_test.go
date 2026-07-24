package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(serverURL string) *RunPodUpscaleWorkerClient {
	return &RunPodUpscaleWorkerClient{
		baseURL:    serverURL,
		apiKey:     "test-key",
		httpClient: &http.Client{},
	}
}

func makeTestJob() UpscaleWorkerJob {
	return UpscaleWorkerJob{
		JobID:            42,
		SourceURL:        "https://example.com/source.mp4",
		SourceStorageKey: "animes/test/source.mp4",
		ResultStorageKey: "animes/test/source_upscaled.mp4",
		TargetHeight:     1080,
	}
}

func TestRunPodClient_Enqueue_Success(t *testing.T) {
	var received runPodRequest
	var gotAuth string
	var gotMethod string
	var gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test-runpod-123","status":"IN_QUEUE"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	runpodID, err := client.Enqueue(context.Background(), makeTestJob())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runpodID != "test-runpod-123" {
		t.Errorf("expected runpod id 'test-runpod-123', got %q", runpodID)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}

	if gotContentType != "application/json" {
		t.Errorf("expected application/json, got %q", gotContentType)
	}

	if gotAuth != "Bearer test-key" {
		t.Errorf("expected 'Bearer test-key', got %q", gotAuth)
	}

	if received.Input.JobID != 42 {
		t.Errorf("expected jobId 42, got %d", received.Input.JobID)
	}

	if received.Input.TargetHeight != 1080 {
		t.Errorf("expected targetHeight 1080, got %d", received.Input.TargetHeight)
	}
}

func TestRunPodClient_Enqueue_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.Enqueue(context.Background(), makeTestJob())

	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestRunPodClient_Status_Completed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected auth header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"job-123","status":"COMPLETED","output":{"resultStorageKey":"animes/test/upscaled.mp4","status":"completed","variantHeights":"","stageTimings":{"model":184.84,"total":203.07}}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	status, err := client.Status(context.Background(), "job-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", status.Status)
	}

	if status.Output.ResultStorageKey != "animes/test/upscaled.mp4" {
		t.Errorf("expected resultStorageKey in output, got %v", status.Output)
	}
}

func TestRunPodClient_Status_Failed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"job-456","status":"FAILED","error":"GPU OOM"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	status, err := client.Status(context.Background(), "job-456")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Status != "FAILED" {
		t.Errorf("expected FAILED, got %s", status.Status)
	}

	if status.Error != "GPU OOM" {
		t.Errorf("expected error 'GPU OOM', got %q", status.Error)
	}
}

func TestRunPodClient_Enqueue_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := newTestClient(server.URL)
	_, err := client.Enqueue(context.Background(), makeTestJob())

	if err == nil {
		t.Fatal("expected error for closed server")
	}
}
