package handler

import (
	"testing"

	"upanime/api/model"
)

func variantEpisode() *model.Episode {
	return &model.Episode{
		StorageKey:         "animes/test/ep1.mp4",
		UpscaledStorageKey: "animes/test/ep1_upscaled.mp4",
		UpscaledVariants: []model.EpisodeVariant{
			{Height: 2160, StorageKey: "animes/test/ep1_upscaled.mp4"},
			{Height: 1440, StorageKey: "animes/test/ep1_upscaled_1440p.mp4"},
			{Height: 1080, StorageKey: "animes/test/ep1_upscaled_1080p.mp4"},
		},
	}
}

func TestEpisodeStorageKeyForVariant(t *testing.T) {
	ep := variantEpisode()
	cases := []struct {
		variant string
		want    string
	}{
		{"", "animes/test/ep1.mp4"},
		{"original", "animes/test/ep1.mp4"},
		{"upscaled", "animes/test/ep1_upscaled.mp4"},
		{"1080p", "animes/test/ep1_upscaled_1080p.mp4"},
		{"1440p", "animes/test/ep1_upscaled_1440p.mp4"},
		{"2160p", "animes/test/ep1_upscaled.mp4"},
		{"1080", "animes/test/ep1_upscaled_1080p.mp4"},
	}
	for _, c := range cases {
		if got := episodeStorageKeyForVariant(ep, c.variant); got != c.want {
			t.Errorf("variant %q: expected %q, got %q", c.variant, c.want, got)
		}
	}
}

func TestEpisodeStorageKeyForVariant_UnknownHeightFallsBackToUpscaled(t *testing.T) {
	ep := variantEpisode()
	if got := episodeStorageKeyForVariant(ep, "720p"); got != ep.UpscaledStorageKey {
		t.Fatalf("expected fallback to upscaled, got %q", got)
	}
}

func TestEpisodeStorageKeyForVariant_HeightWithoutVariantsFallsBackToOriginal(t *testing.T) {
	ep := &model.Episode{StorageKey: "animes/test/ep1.mp4"}
	if got := episodeStorageKeyForVariant(ep, "1080p"); got != ep.StorageKey {
		t.Fatalf("expected fallback to original when no upscaled exists, got %q", got)
	}
}

func TestParseVariantHeight(t *testing.T) {
	cases := []struct {
		in     string
		height int
		ok     bool
	}{
		{"1080p", 1080, true},
		{"1440", 1440, true},
		{"original", 0, false},
		{"upscaled", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		height, ok := parseVariantHeight(c.in)
		if ok != c.ok || height != c.height {
			t.Errorf("parseVariantHeight(%q) = (%d,%v), want (%d,%v)", c.in, height, ok, c.height, c.ok)
		}
	}
}
