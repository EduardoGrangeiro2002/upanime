package storage_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"upanime/api/storage"
)

func r2Config(t *testing.T) (accountID, keyID, secret, bucket string) {
	t.Helper()
	accountID = os.Getenv("R2_ACCOUNT_ID")
	keyID = os.Getenv("R2_ACCESS_KEY_ID")
	secret = os.Getenv("R2_ACCESS_SECRET")
	bucket = os.Getenv("R2_BUCKET_NAME")

	if accountID == "" || keyID == "" || secret == "" || bucket == "" {
		t.Skip("R2 env vars not set, skipping integration test")
	}
	return
}

func TestR2Storage_SaveExistsDeleteURL(t *testing.T) {
	accountID, keyID, secret, bucket := r2Config(t)
	s := storage.NewR2Storage(accountID, keyID, secret, bucket)
	ctx := context.Background()

	key := "test/integration_test.txt"

	err := s.Save(ctx, key, strings.NewReader("hello r2"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err := s.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected file to exist after Save")
	}

	url, err := s.URL(ctx, key)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.Contains(url, key) {
		t.Errorf("expected URL to contain key, got: %s", url)
	}

	err = s.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, _ = s.Exists(ctx, key)
	if exists {
		t.Fatal("expected file to not exist after Delete")
	}
}

func TestR2Storage_ListKeys(t *testing.T) {
	accountID, keyID, secret, bucket := r2Config(t)
	s := storage.NewR2Storage(accountID, keyID, secret, bucket)
	ctx := context.Background()

	prefix := "test/listkeys/"
	_ = s.Save(ctx, prefix+"a.txt", strings.NewReader("a"))
	_ = s.Save(ctx, prefix+"b.txt", strings.NewReader("b"))

	keys, err := s.ListKeys(ctx, prefix)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys, got %d", len(keys))
	}

	_ = s.Delete(ctx, prefix+"a.txt")
	_ = s.Delete(ctx, prefix+"b.txt")
}
