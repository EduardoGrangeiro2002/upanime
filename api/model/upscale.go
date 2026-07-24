package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

type EpisodeVariant struct {
	Height     int    `json:"height"`
	StorageKey string `json:"storageKey"`
}

var variantLadder = []int{1440, 1080}

func VariantHeights(targetHeight int) []int {
	heights := make([]int, 0, len(variantLadder))
	for _, h := range variantLadder {
		if h >= targetHeight {
			continue
		}
		heights = append(heights, h)
	}
	return heights
}

func BuildVariantKey(resultKey string, height int) string {
	ext := filepath.Ext(resultKey)
	base := strings.TrimSuffix(resultKey, ext)
	return fmt.Sprintf("%s_%dp%s", base, height, ext)
}

type UpscaleJob struct {
	ID               StringID `json:"id"`
	EpisodeID        StringID `json:"episodeId"`
	AnimeID          StringID `json:"animeId"`
	Type             string   `json:"type"`
	TargetHeight     int      `json:"targetHeight,omitempty"`
	BatchSize        *int     `json:"batchSize,omitempty"`
	Sharpen          *float64 `json:"sharpen,omitempty"`
	Saturation       *float64 `json:"saturation,omitempty"`
	Contrast         *float64 `json:"contrast,omitempty"`
	Interpolate      bool     `json:"interpolate,omitempty"`
	PanRatio         *float64 `json:"panRatio,omitempty"`
	Effects          bool     `json:"effects,omitempty"`
	EffectsStrength  *float64 `json:"effectsStrength,omitempty"`
	EffectsSens      *float64 `json:"effectsSensitivity,omitempty"`
	SkipUpscale      bool     `json:"skipUpscale,omitempty"`
	Upscaler         string   `json:"upscaler,omitempty"`
	RunPodJobID      string   `json:"-"`
	SourceStorageKey string   `json:"-"`
	ResultStorageKey string   `json:"resultStorageKey,omitempty"`
	Status           string   `json:"status"`
	Error            string   `json:"error,omitempty"`
	AnimeTitle       string   `json:"animeTitle"`
	EpisodeTitle     string   `json:"episodeTitle"`
	EpisodeNumber    string   `json:"episodeNumber"`
	SeasonNumber     int      `json:"seasonNumber"`
	AnimeImageURL    string   `json:"animeImageUrl"`
	CreatedAt        string   `json:"createdAt,omitempty"`
	UpdatedAt        string   `json:"updatedAt,omitempty"`
}

type CreateUpscaleRequest struct {
	AnimeID         StringID   `json:"animeId"`
	EpisodeIDs      []StringID `json:"episodeIds"`
	TargetHeight    int        `json:"targetHeight,omitempty"`
	BatchSize       *int       `json:"batchSize,omitempty"`
	Sharpen         *float64   `json:"sharpen,omitempty"`
	Saturation      *float64   `json:"saturation,omitempty"`
	Contrast        *float64   `json:"contrast,omitempty"`
	Interpolate     bool       `json:"interpolate,omitempty"`
	PanRatio        *float64   `json:"panRatio,omitempty"`
	Effects         bool       `json:"effects,omitempty"`
	EffectsStrength *float64   `json:"effectsStrength,omitempty"`
	EffectsSens     *float64   `json:"effectsSensitivity,omitempty"`
	SkipUpscale     bool       `json:"skipUpscale,omitempty"`
	Upscaler        string     `json:"upscaler,omitempty"`
}
