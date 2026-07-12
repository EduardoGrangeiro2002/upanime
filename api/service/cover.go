package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"upanime/api/storage"
)

func DownloadCover(ctx context.Context, imageURL string, animeSlug string, fs storage.FileStorage) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download cover: status %d", resp.StatusCode)
	}

	ext := extensionFromContentType(resp.Header.Get("Content-Type"))
	key := fmt.Sprintf("animes/%s/cover%s", animeSlug, ext)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read cover body: %w", err)
	}

	if err := fs.Save(ctx, key, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("save cover: %w", err)
	}

	return key, nil
}

func extensionFromContentType(ct string) string {
	mediaType := strings.SplitN(ct, ";", 2)[0]
	mediaType = strings.TrimSpace(mediaType)

	exts, err := mime.ExtensionsByType(mediaType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}

	switch mediaType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	}

	return ".jpg"
}
