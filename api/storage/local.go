package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

func (s *LocalStorage) Save(_ context.Context, key string, reader io.Reader) error {
	fullPath := filepath.Join(s.basePath, key)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}

func (s *LocalStorage) Download(_ context.Context, key string, destPath string) error {
	srcPath := filepath.Join(s.basePath, key)

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(s.basePath, key))
}

func (s *LocalStorage) URL(_ context.Context, key string) (string, error) {
	return filepath.Join(s.basePath, key), nil
}

func (s *LocalStorage) PresignPutURL(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("presigned PUT URLs are not supported with local storage")
}

func (s *LocalStorage) Exists(_ context.Context, key string) (bool, error) {
	_, err := os.Stat(filepath.Join(s.basePath, key))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *LocalStorage) ServeFile(_ context.Context, w http.ResponseWriter, r *http.Request, key string) error {
	http.ServeFile(w, r, filepath.Join(s.basePath, key))
	return nil
}

func (s *LocalStorage) ListKeys(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	root := filepath.Join(s.basePath, prefix)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return err
		}
		keys = append(keys, strings.ReplaceAll(rel, string(filepath.Separator), "/"))
		return nil
	})

	if os.IsNotExist(err) {
		return keys, nil
	}

	return keys, err
}
