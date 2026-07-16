package service

import (
	"context"
)

type WorkerVariant struct {
	Height     int    `json:"height"`
	StorageKey string `json:"storageKey"`
}

type UpscaleWorkerJob struct {
	JobID            int64           `json:"jobId"`
	SourceURL        string          `json:"sourceUrl"`
	SourceStorageKey string          `json:"sourceStorageKey"`
	ResultStorageKey string          `json:"resultStorageKey"`
	Variants         []WorkerVariant `json:"variants,omitempty"`
	TargetHeight     int             `json:"targetHeight,omitempty"`
	BatchSize        *int            `json:"batchSize,omitempty"`
	Sharpen          *float64        `json:"sharpen,omitempty"`
	Saturation       *float64        `json:"saturation,omitempty"`
	Contrast         *float64        `json:"contrast,omitempty"`
	Interpolate      bool            `json:"interpolate,omitempty"`
	PanRatio         *float64        `json:"panRatio,omitempty"`
	Effects          bool            `json:"effects,omitempty"`
	EffectsStrength  *float64        `json:"effectsStrength,omitempty"`
	EffectsSens      *float64        `json:"effectsSensitivity,omitempty"`
	SkipUpscale      bool            `json:"skipUpscale,omitempty"`
	CallbackURL      string          `json:"callbackUrl,omitempty"`
}

type RunPodJobStatus struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	Output        map[string]string `json:"output,omitempty"`
	Error         string            `json:"error,omitempty"`
	DelayTime     int               `json:"delayTime,omitempty"`
	ExecutionTime int               `json:"executionTime,omitempty"`
}

type UpscaleWorkerClient interface {
	Enqueue(ctx context.Context, job UpscaleWorkerJob) (string, error)
	Status(ctx context.Context, runpodJobID string) (*RunPodJobStatus, error)
}
