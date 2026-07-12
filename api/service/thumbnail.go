package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"upanime/api/storage"
)

const (
	thumbnailSuffix     = "_thumb.jpg"
	thumbnailWidth      = 480
	extractTimeout      = 60 * time.Second
	maxConcurrentThumbs = 2
)

type FrameExtractor func(ctx context.Context, sourceURL string) ([]byte, error)

type ThumbnailService struct {
	storage   storage.FileStorage
	extract   FrameExtractor
	semaphore chan struct{}
	mu        sync.Mutex
	inFlight  map[string]*sync.Mutex
}

func NewThumbnailService(fs storage.FileStorage, extract FrameExtractor) *ThumbnailService {
	if extract == nil {
		extract = ExtractMiddleFrame
	}
	return &ThumbnailService{
		storage:   fs,
		extract:   extract,
		semaphore: make(chan struct{}, maxConcurrentThumbs),
		inFlight:  make(map[string]*sync.Mutex),
	}
}

func ThumbnailKey(sourceKey string) string {
	ext := filepath.Ext(sourceKey)
	if ext == "" {
		return sourceKey + thumbnailSuffix
	}
	return strings.TrimSuffix(sourceKey, ext) + thumbnailSuffix
}

func (s *ThumbnailService) Ensure(ctx context.Context, sourceKey string) (string, error) {
	thumbKey := ThumbnailKey(sourceKey)

	exists, err := s.storage.Exists(ctx, thumbKey)
	if err == nil && exists {
		return thumbKey, nil
	}

	lock := s.keyLock(thumbKey)
	lock.Lock()
	defer lock.Unlock()

	exists, err = s.storage.Exists(ctx, thumbKey)
	if err == nil && exists {
		return thumbKey, nil
	}

	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	sourceURL, err := s.storage.URL(ctx, sourceKey)
	if err != nil {
		return "", fmt.Errorf("thumbnail source url: %w", err)
	}

	frame, err := s.extract(ctx, sourceURL)
	if err != nil {
		return "", fmt.Errorf("extract frame: %w", err)
	}
	if len(frame) == 0 {
		return "", fmt.Errorf("extract frame: empty output")
	}

	if err := s.storage.Save(ctx, thumbKey, bytes.NewReader(frame)); err != nil {
		return "", fmt.Errorf("save thumbnail: %w", err)
	}
	return thumbKey, nil
}

func (s *ThumbnailService) keyLock(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, ok := s.inFlight[key]
	if !ok {
		lock = &sync.Mutex{}
		s.inFlight[key] = lock
	}
	return lock
}

func ExtractMiddleFrame(ctx context.Context, sourceURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, extractTimeout)
	defer cancel()

	duration, err := probeDuration(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	middle := duration / 2
	command := exec.CommandContext(ctx,
		"ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-ss", strconv.FormatFloat(middle, 'f', 3, 64),
		"-i", sourceURL,
		"-vf", fmt.Sprintf("thumbnail=30,scale=%d:-2", thumbnailWidth),
		"-frames:v", "1",
		"-q:v", "4",
		"-f", "image2", "-",
	)

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func probeDuration(ctx context.Context, sourceURL string) (float64, error) {
	command := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		sourceURL,
	)
	output, err := command.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var probe struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &probe); err != nil {
		return 0, fmt.Errorf("ffprobe parse: %w", err)
	}

	duration, err := strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w", err)
	}
	return duration, nil
}
