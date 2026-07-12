package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	"upanime/api/db"
)

func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	t.Cleanup(func() { database.Close() })
	return database
}
