package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type runPodRequest struct {
	Input UpscaleWorkerJob `json:"input"`
}

type runPodEnqueueResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type RunPodUpscaleWorkerClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewRunPodUpscaleWorkerClient(endpointID, apiKey string) *RunPodUpscaleWorkerClient {
	return &RunPodUpscaleWorkerClient{
		baseURL: fmt.Sprintf("https://api.runpod.ai/v2/%s", endpointID),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *RunPodUpscaleWorkerClient) Enqueue(ctx context.Context, job UpscaleWorkerJob) (string, error) {
	payload := runPodRequest{Input: job}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal runpod request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/run", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create runpod request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("enqueue runpod job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("enqueue runpod job: status %d", resp.StatusCode)
	}

	var result runPodEnqueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode runpod response: %w", err)
	}

	return result.ID, nil
}

func (c *RunPodUpscaleWorkerClient) Status(ctx context.Context, runpodJobID string) (*RunPodJobStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status/"+runpodJobID, nil)
	if err != nil {
		return nil, fmt.Errorf("create runpod status request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get runpod status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get runpod status: status %d body %s", resp.StatusCode, string(body))
	}

	var status RunPodJobStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode runpod status: %w", err)
	}

	return &status, nil
}
