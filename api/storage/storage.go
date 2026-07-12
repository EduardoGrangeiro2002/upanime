package storage

import (
	"context"
	"io"
	"net/http"
)

type FileStorage interface {
	Save(ctx context.Context, key string, reader io.Reader) error
	Download(ctx context.Context, key string, destPath string) error
	Delete(ctx context.Context, key string) error
	URL(ctx context.Context, key string) (string, error)
	PresignPutURL(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	ListKeys(ctx context.Context, prefix string) ([]string, error)
	ServeFile(ctx context.Context, w http.ResponseWriter, r *http.Request, key string) error
}
