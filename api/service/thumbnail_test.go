package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"upanime/api/storage"
)

func TestThumbnailKey(t *testing.T) {
	cases := map[string]string{
		"animes/naruto/S1E1/ep.mp4": "animes/naruto/S1E1/ep_thumb.jpg",
		"animes/one-piece/ep.mkv":   "animes/one-piece/ep_thumb.jpg",
		"sem-extensao":              "sem-extensao_thumb.jpg",
	}
	for source, expected := range cases {
		if got := ThumbnailKey(source); got != expected {
			t.Fatalf("ThumbnailKey(%q) = %q, expected %q", source, got, expected)
		}
	}
}

func TestEnsureGeneratesOnceAndCaches(t *testing.T) {
	fs := storage.NewLocalStorage(t.TempDir())
	var calls atomic.Int32
	extractor := func(_ context.Context, _ string) ([]byte, error) {
		calls.Add(1)
		return []byte("fake-jpeg"), nil
	}
	service := NewThumbnailService(fs, extractor)

	sourceKey := "animes/test/ep.mp4"
	first, err := service.Ensure(context.Background(), sourceKey)
	if err != nil {
		t.Fatal(err)
	}
	if first != "animes/test/ep_thumb.jpg" {
		t.Fatalf("unexpected thumb key: %s", first)
	}

	second, err := service.Ensure(context.Background(), sourceKey)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("expected same key, got %s", second)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 extraction, got %d", calls.Load())
	}
}

func TestEnsureConcurrentRequestsExtractOnce(t *testing.T) {
	fs := storage.NewLocalStorage(t.TempDir())
	var calls atomic.Int32
	extractor := func(_ context.Context, _ string) ([]byte, error) {
		calls.Add(1)
		return []byte("fake-jpeg"), nil
	}
	service := NewThumbnailService(fs, extractor)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.Ensure(context.Background(), "animes/test/ep.mp4"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("expected 1 extraction under concurrency, got %d", calls.Load())
	}
}

func TestEnsureFailsOnEmptyFrame(t *testing.T) {
	fs := storage.NewLocalStorage(t.TempDir())
	extractor := func(_ context.Context, _ string) ([]byte, error) {
		return nil, nil
	}
	service := NewThumbnailService(fs, extractor)

	if _, err := service.Ensure(context.Background(), "animes/test/ep.mp4"); err == nil {
		t.Fatal("expected error for empty frame")
	}
}

func TestExtractMiddleFrameWithRealFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "sample.mp4")
	generate := exec.Command(
		"ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=12",
		"-pix_fmt", "yuv420p", video,
	)
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate sample video: %v: %s", err, output)
	}

	frame, err := ExtractMiddleFrame(context.Background(), video)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) < 2 || frame[0] != 0xFF || frame[1] != 0xD8 {
		t.Fatalf("expected JPEG magic bytes, got %d bytes", len(frame))
	}

	if err := os.WriteFile(filepath.Join(dir, "thumb.jpg"), frame, 0o644); err != nil {
		t.Fatal(err)
	}
}
