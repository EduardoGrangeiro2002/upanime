package storage

import "testing"

func TestContentTypeForKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"animes/test/ep.mp4", "video/mp4"},
		{"animes/Dragon_Ball_Z_Dublado/Episódio_001.mp4", "video/mp4"},
		{"animes/test/cover.jpeg", "image/jpeg"},
		{"animes/test/cover.jpg", "image/jpeg"},
		{"animes/test/cover.png", "image/png"},
		{"animes/test/cover.webp", "image/webp"},
		{"animes/test/video.webm", "video/webm"},
		{"animes/test/video.mkv", "video/x-matroska"},
		{"animes/test/ep_upscaled.MP4", "video/mp4"},
		{"animes/test/unknown.zzz", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
	}

	for _, tt := range tests {
		got := contentTypeForKey(tt.key)
		if got != tt.want {
			t.Errorf("contentTypeForKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
