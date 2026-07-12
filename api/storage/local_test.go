package storage_test

import (
	"context"
	"strings"
	"testing"

	"upanime/api/storage"
)

func TestLocalStorage_SaveAndExists(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)
	ctx := context.Background()

	err := s.Save(ctx, "test/file.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err := s.Exists(ctx, "test/file.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected file to exist")
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)
	ctx := context.Background()

	_ = s.Save(ctx, "del.txt", strings.NewReader("data"))

	err := s.Delete(ctx, "del.txt")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, _ := s.Exists(ctx, "del.txt")
	if exists {
		t.Fatal("expected file to be deleted")
	}
}

func TestLocalStorage_URL(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)
	ctx := context.Background()

	url, err := s.URL(ctx, "some/path.mp4")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.Contains(url, "some/path.mp4") {
		t.Fatalf("unexpected URL: %s", url)
	}
}

func TestLocalStorage_ListKeys(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)
	ctx := context.Background()

	_ = s.Save(ctx, "anime/ep1.mp4", strings.NewReader("data1"))
	_ = s.Save(ctx, "anime/ep2.mp4", strings.NewReader("data2"))
	_ = s.Save(ctx, "other/file.txt", strings.NewReader("data3"))

	keys, err := s.ListKeys(ctx, "anime")
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestLocalStorage_ListKeys_EmptyPrefix(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewLocalStorage(dir)
	ctx := context.Background()

	keys, err := s.ListKeys(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}
