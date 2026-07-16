package model

import (
	"reflect"
	"testing"
)

func TestVariantHeights(t *testing.T) {
	cases := []struct {
		target int
		want   []int
	}{
		{2160, []int{1440, 1080}},
		{1440, []int{1080}},
		{1080, []int{}},
	}

	for _, c := range cases {
		got := VariantHeights(c.target)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("VariantHeights(%d) = %v, want %v", c.target, got, c.want)
		}
	}
}

func TestBuildVariantKey(t *testing.T) {
	got := BuildVariantKey("animes/slayers/s01e04_upscaled.mp4", 1440)
	want := "animes/slayers/s01e04_upscaled_1440p.mp4"
	if got != want {
		t.Fatalf("BuildVariantKey = %q, want %q", got, want)
	}
}

func TestBuildVariantKeyWithoutExtension(t *testing.T) {
	got := BuildVariantKey("animes/slayers/s01e04_upscaled", 1080)
	want := "animes/slayers/s01e04_upscaled_1080p"
	if got != want {
		t.Fatalf("BuildVariantKey = %q, want %q", got, want)
	}
}
