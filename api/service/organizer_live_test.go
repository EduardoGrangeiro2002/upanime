package service

import (
	"context"
	"os"
	"testing"
)

func TestRenumber_LiveOpenRouter(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}

	o := NewEpisodeOrganizer(apiKey, "", "")
	titles := []string{
		"Dragon Ball (Dublado) – Dublado – Episódio 01 – O Segredo das Esferas do Dragão",
		"Dragon Ball (Dublado) – Dublado – Episódio 153 – A Montanha Frypan Está em Chamas!",
		"Dragon Ball – Filme 1 – A Lenda de Shenlong",
		"Naruto Shippuden Episódio 500 (Final)",
	}

	numbers, err := o.Renumber(context.Background(), titles)
	if err != nil {
		t.Fatalf("live renumber: %v", err)
	}

	want := []string{"1", "153", "", "500"}
	for i, n := range numbers {
		if n != want[i] {
			t.Errorf("position %d: expected %q, got %q (title %q)", i, want[i], n, titles[i])
		}
	}
}
