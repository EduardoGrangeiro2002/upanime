package handler

import (
	"testing"

	"upanime/api/model"
)

func TestBuildWorkerVariants(t *testing.T) {
	t.Parallel()

	job := model.UpscaleJob{
		TargetHeight:     2160,
		ResultStorageKey: "animes/naruto/ep_1_upscaled.mp4",
	}

	variants := buildWorkerVariants(job)

	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if variants[0].Height != 1440 || variants[0].StorageKey != "animes/naruto/ep_1_upscaled_1440p.mp4" {
		t.Fatalf("unexpected first variant: %+v", variants[0])
	}
	if variants[1].Height != 1080 || variants[1].StorageKey != "animes/naruto/ep_1_upscaled_1080p.mp4" {
		t.Fatalf("unexpected second variant: %+v", variants[1])
	}
}

func TestBuildWorkerVariantsFor1080IsEmpty(t *testing.T) {
	t.Parallel()

	job := model.UpscaleJob{
		TargetHeight:     1080,
		ResultStorageKey: "animes/naruto/ep_1_upscaled.mp4",
	}

	if got := buildWorkerVariants(job); len(got) != 0 {
		t.Fatalf("expected no variants, got %+v", got)
	}
}

func TestBuildWorkerVariantsSkipUpscale(t *testing.T) {
	t.Parallel()

	job := model.UpscaleJob{
		TargetHeight:     2160,
		ResultStorageKey: "animes/naruto/ep_1_upscaled.mp4",
		SkipUpscale:      true,
	}

	if got := buildWorkerVariants(job); got != nil {
		t.Fatalf("expected nil variants, got %+v", got)
	}
}

func TestBuildWorkerJobLeavesSourceURLEmpty(t *testing.T) {
	t.Parallel()

	sharpen := 0.5
	job := model.UpscaleJob{
		ID:               model.StringID(9),
		TargetHeight:     1440,
		Sharpen:          &sharpen,
		Interpolate:      true,
		Upscaler:         "apisr",
		SourceStorageKey: "animes/naruto/ep_1.mp4",
		ResultStorageKey: "animes/naruto/ep_1_upscaled.mp4",
	}

	wj := buildWorkerJob(job)

	if wj.JobID != 9 {
		t.Fatalf("expected job id 9, got %d", wj.JobID)
	}
	if wj.SourceURL != "" {
		t.Fatalf("expected empty source url, got %s", wj.SourceURL)
	}
	if wj.SourceStorageKey != job.SourceStorageKey || wj.ResultStorageKey != job.ResultStorageKey {
		t.Fatal("expected storage keys carried to worker job")
	}
	if wj.Sharpen == nil || *wj.Sharpen != 0.5 || !wj.Interpolate {
		t.Fatal("expected effect params carried to worker job")
	}
	if wj.Upscaler != "apisr" {
		t.Fatalf("expected upscaler carried to worker job, got %q", wj.Upscaler)
	}
	if len(wj.Variants) != 1 || wj.Variants[0].Height != 1080 {
		t.Fatalf("expected 1080 variant, got %+v", wj.Variants)
	}
}

func TestBuildUpscaledKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sourceKey string
		want      string
	}{
		{
			name:      "adds suffix before extension",
			sourceKey: "animes/naruto/ep_1.mp4",
			want:      "animes/naruto/ep_1_upscaled.mp4",
		},
		{
			name:      "handles files without extension",
			sourceKey: "animes/naruto/ep_1",
			want:      "animes/naruto/ep_1_upscaled",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := buildUpscaledKey(test.sourceKey)
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestNormalizeTargetHeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{name: "defaults to 1080", input: 0, want: 1080},
		{name: "1080 is valid", input: 1080, want: 1080},
		{name: "1440 is valid", input: 1440, want: 1440},
		{name: "2160 is valid", input: 2160, want: 2160},
		{name: "invalid height fails", input: 720, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeTargetHeight(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %d, got %d", test.want, got)
			}
		})
	}
}
