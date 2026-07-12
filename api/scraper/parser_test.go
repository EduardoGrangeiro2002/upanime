package scraper

import (
	"encoding/json"
	"testing"
)

func TestParseScrapeOutput_Valid(t *testing.T) {
	input := ScrapeAnimeOutput{
		Title:       "Naruto",
		URL:         "https://example.com/naruto",
		ImageURL:    "https://example.com/naruto.jpg",
		Description: "Ninja anime",
		Episodes: []ScrapeEpisodeEntry{
			{Title: "Ep 1", URL: "https://example.com/ep1", Season: "1", Number: "1"},
			{Title: "Ep 2", URL: "https://example.com/ep2", Season: "1", Number: "2"},
			{Title: "Ep 1 S2", URL: "https://example.com/s2ep1", Season: "2", Number: "1"},
		},
	}

	data, _ := json.Marshal(input)
	anime, err := ParseScrapeOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if anime.Title != "Naruto" {
		t.Errorf("expected title 'Naruto', got '%s'", anime.Title)
	}
	if len(anime.Seasons) != 2 {
		t.Fatalf("expected 2 seasons, got %d", len(anime.Seasons))
	}
	if len(anime.Seasons[0].Episodes) != 2 {
		t.Errorf("expected 2 episodes in season 1, got %d", len(anime.Seasons[0].Episodes))
	}
	if len(anime.Seasons[1].Episodes) != 1 {
		t.Errorf("expected 1 episode in season 2, got %d", len(anime.Seasons[1].Episodes))
	}
}

func TestParseScrapeOutput_InvalidJSON(t *testing.T) {
	_, err := ParseScrapeOutput([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseScrapeOutput_MissingTitle(t *testing.T) {
	input := ScrapeAnimeOutput{
		URL:      "https://example.com/anime",
		Episodes: []ScrapeEpisodeEntry{{Title: "Ep 1", URL: "https://example.com/ep1", Season: "1", Number: "1"}},
	}
	data, _ := json.Marshal(input)

	_, err := ParseScrapeOutput(data)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseScrapeOutput_MissingURL(t *testing.T) {
	input := ScrapeAnimeOutput{
		Title:    "Naruto",
		Episodes: []ScrapeEpisodeEntry{{Title: "Ep 1", URL: "https://example.com/ep1", Season: "1", Number: "1"}},
	}
	data, _ := json.Marshal(input)

	_, err := ParseScrapeOutput(data)
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestParseScrapeOutput_NoEpisodes(t *testing.T) {
	input := ScrapeAnimeOutput{
		Title:    "Naruto",
		URL:      "https://example.com/naruto",
		Episodes: []ScrapeEpisodeEntry{},
	}
	data, _ := json.Marshal(input)

	_, err := ParseScrapeOutput(data)
	if err == nil {
		t.Fatal("expected error for no episodes")
	}
}

func TestValidate_Valid(t *testing.T) {
	o := &ScrapeAnimeOutput{
		Title:    "Test",
		URL:      "https://example.com",
		Episodes: []ScrapeEpisodeEntry{{Title: "Ep", URL: "https://example.com/ep", Season: "1", Number: "1"}},
	}
	if err := o.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyTitle(t *testing.T) {
	o := &ScrapeAnimeOutput{
		URL:      "https://example.com",
		Episodes: []ScrapeEpisodeEntry{{Title: "Ep", URL: "https://example.com/ep", Season: "1", Number: "1"}},
	}
	if err := o.Validate(); err == nil {
		t.Error("expected error for empty title")
	}
}

func TestValidate_EmptyURL(t *testing.T) {
	o := &ScrapeAnimeOutput{
		Title:    "Test",
		Episodes: []ScrapeEpisodeEntry{{Title: "Ep", URL: "https://example.com/ep", Season: "1", Number: "1"}},
	}
	if err := o.Validate(); err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestValidate_NoEpisodes(t *testing.T) {
	o := &ScrapeAnimeOutput{
		Title: "Test",
		URL:   "https://example.com",
	}
	if err := o.Validate(); err == nil {
		t.Error("expected error for no episodes")
	}
}
