package handler

import "testing"

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
