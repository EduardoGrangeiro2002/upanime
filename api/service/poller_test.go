package service

import (
	"context"
	"testing"

	"upanime/api/model"
)

type fakeEpisodeStore struct {
	savedVariants []model.EpisodeVariant
	savedID       int64
}

func (f *fakeEpisodeStore) GetByID(_ context.Context, _ int64) (*model.Episode, error) {
	return nil, nil
}

func (f *fakeEpisodeStore) Delete(_ context.Context, _ int64) error { return nil }

func (f *fakeEpisodeStore) UpdateStorageKey(_ context.Context, _ int64, _ string) error { return nil }

func (f *fakeEpisodeStore) UpdateUpscaledStorageKey(_ context.Context, _ int64, _ string) error {
	return nil
}

func (f *fakeEpisodeStore) UpdateUpscaledVariants(_ context.Context, id int64, variants []model.EpisodeVariant) error {
	f.savedID = id
	f.savedVariants = variants
	return nil
}

func TestSaveVariantsIncludesTargetAndConfirmedHeights(t *testing.T) {
	episodes := &fakeEpisodeStore{}
	poller := &RunPodPoller{episodes: episodes}
	job := model.UpscaleJob{
		EpisodeID:    model.StringID(7),
		TargetHeight: 2160,
	}

	poller.saveVariants(context.Background(), job, "animes/a/ep_upscaled.mp4", "1440,1080")

	want := []model.EpisodeVariant{
		{Height: 2160, StorageKey: "animes/a/ep_upscaled.mp4"},
		{Height: 1440, StorageKey: "animes/a/ep_upscaled_1440p.mp4"},
		{Height: 1080, StorageKey: "animes/a/ep_upscaled_1080p.mp4"},
	}
	if episodes.savedID != 7 {
		t.Fatalf("expected episode 7, got %d", episodes.savedID)
	}
	if len(episodes.savedVariants) != len(want) {
		t.Fatalf("expected %d variants, got %+v", len(want), episodes.savedVariants)
	}
	for i, v := range want {
		if episodes.savedVariants[i] != v {
			t.Fatalf("variant %d = %+v, want %+v", i, episodes.savedVariants[i], v)
		}
	}
}

func TestSaveVariantsWithEmptyHeightsSavesOnlyTarget(t *testing.T) {
	episodes := &fakeEpisodeStore{}
	poller := &RunPodPoller{episodes: episodes}
	job := model.UpscaleJob{
		EpisodeID:    model.StringID(3),
		TargetHeight: 1080,
	}

	poller.saveVariants(context.Background(), job, "animes/a/ep_upscaled.mp4", "")

	if len(episodes.savedVariants) != 1 {
		t.Fatalf("expected 1 variant, got %+v", episodes.savedVariants)
	}
	if episodes.savedVariants[0].Height != 1080 {
		t.Fatalf("unexpected variant: %+v", episodes.savedVariants[0])
	}
}

func TestSaveVariantsSkipUpscaleDoesNothing(t *testing.T) {
	episodes := &fakeEpisodeStore{}
	poller := &RunPodPoller{episodes: episodes}
	job := model.UpscaleJob{
		EpisodeID:    model.StringID(3),
		TargetHeight: 2160,
		SkipUpscale:  true,
	}

	poller.saveVariants(context.Background(), job, "animes/a/ep_preview.mp4", "1440")

	if episodes.savedVariants != nil {
		t.Fatalf("expected no variants saved, got %+v", episodes.savedVariants)
	}
}
