package store_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"upanime/api/db"
	"upanime/api/model"
	"upanime/api/store"
)

func newDatasetStore(t *testing.T) *store.SQLiteDatasetStore {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "ml_dataset.db"))
	if err != nil {
		t.Fatalf("open dataset db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	datasetStore, err := store.NewSQLiteDatasetStore(database)
	if err != nil {
		t.Fatalf("create dataset store: %v", err)
	}
	return datasetStore
}

func createSample(t *testing.T, s *store.SQLiteDatasetStore, class string) *model.DatasetSample {
	t.Helper()

	sample := &model.DatasetSample{
		Class:       class,
		FrameKey:    "ml-dataset/frames/abc.jpg",
		MaskKey:     "ml-dataset/masks/abc.png",
		AnimeTitle:  "Slayers",
		Episode:     "S1E04",
		TimestampS:  54.3,
		TeacherProb: 0.42,
	}
	if err := s.Create(t.Context(), sample); err != nil {
		t.Fatalf("create sample: %v", err)
	}
	return sample
}

func TestDatasetStore_CreateDefaults(t *testing.T) {
	s := newDatasetStore(t)

	sample := createSample(t, s, "fire")

	if sample.ID.Int64() == 0 {
		t.Error("expected id to be set")
	}
	if sample.Status != "pending" {
		t.Errorf("expected status pending, got %s", sample.Status)
	}
	if sample.Source != "teacher" {
		t.Errorf("expected default source teacher, got %s", sample.Source)
	}
}

func TestDatasetStore_QueueReturnsPendingInOrder(t *testing.T) {
	s := newDatasetStore(t)
	first := createSample(t, s, "fire")
	second := createSample(t, s, "lightning")
	third := createSample(t, s, "aura")

	if err := s.SetVerdict(t.Context(), second.ID.Int64(), "approved"); err != nil {
		t.Fatalf("set verdict: %v", err)
	}

	queue, err := s.Queue(t.Context(), 10)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}

	if len(queue) != 2 {
		t.Fatalf("expected 2 pending samples, got %d", len(queue))
	}
	if queue[0].ID != first.ID || queue[1].ID != third.ID {
		t.Errorf("unexpected queue order: %v, %v", queue[0].ID, queue[1].ID)
	}
}

func TestDatasetStore_QueueRespectsLimit(t *testing.T) {
	s := newDatasetStore(t)
	for range 5 {
		createSample(t, s, "fire")
	}

	queue, err := s.Queue(t.Context(), 3)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if len(queue) != 3 {
		t.Errorf("expected 3 samples, got %d", len(queue))
	}
}

func TestDatasetStore_SetVerdictUnknownID(t *testing.T) {
	s := newDatasetStore(t)

	err := s.SetVerdict(t.Context(), 999, "approved")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDatasetStore_SetVerdictStampsReviewedAt(t *testing.T) {
	s := newDatasetStore(t)
	sample := createSample(t, s, "fire")

	if err := s.SetVerdict(t.Context(), sample.ID.Int64(), "needs_edit"); err != nil {
		t.Fatalf("set verdict: %v", err)
	}

	stats, err := s.Stats(t.Context())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 1 || stats[0].Status != "needs_edit" || stats[0].Count != 1 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}

func TestDatasetStore_StatsGroupsByClassAndStatus(t *testing.T) {
	s := newDatasetStore(t)
	createSample(t, s, "fire")
	fire := createSample(t, s, "fire")
	lightning := createSample(t, s, "lightning")

	if err := s.SetVerdict(t.Context(), fire.ID.Int64(), "approved"); err != nil {
		t.Fatalf("set verdict: %v", err)
	}
	if err := s.SetVerdict(t.Context(), lightning.ID.Int64(), "rejected"); err != nil {
		t.Fatalf("set verdict: %v", err)
	}

	stats, err := s.Stats(t.Context())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	counts := map[string]int{}
	for _, stat := range stats {
		counts[stat.Class+"/"+stat.Status] = stat.Count
	}
	if counts["fire/pending"] != 1 || counts["fire/approved"] != 1 || counts["lightning/rejected"] != 1 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}
