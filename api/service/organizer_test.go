package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"upanime/api/model"
)

func fakeOpenRouter(t *testing.T, reply string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": reply}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRenumber_NormalizesNumbers(t *testing.T) {
	server := fakeOpenRouter(t, `["01", "2", "", "abc"]`)
	defer server.Close()

	o := NewEpisodeOrganizer("test-key", "test-model", server.URL)
	numbers, err := o.Renumber(context.Background(), []string{"a", "b", "c", "d"})
	if err != nil {
		t.Fatalf("renumber: %v", err)
	}

	want := []string{"1", "2", "", ""}
	for i, n := range numbers {
		if n != want[i] {
			t.Errorf("position %d: expected %q, got %q", i, want[i], n)
		}
	}
}

func TestRenumber_LengthMismatch(t *testing.T) {
	server := fakeOpenRouter(t, `["1", "2"]`)
	defer server.Close()

	o := NewEpisodeOrganizer("test-key", "test-model", server.URL)
	_, err := o.Renumber(context.Background(), []string{"a", "b", "c"})
	if err == nil {
		t.Fatal("expected length mismatch error")
	}
}

func TestRenumber_EmptyTitles(t *testing.T) {
	o := NewEpisodeOrganizer("test-key", "test-model", "http://unused")
	numbers, err := o.Renumber(context.Background(), nil)
	if err != nil {
		t.Fatalf("renumber: %v", err)
	}
	if len(numbers) != 0 {
		t.Errorf("expected empty result, got %v", numbers)
	}
}

func TestOrganizeAnime_UpdatesChangedNumbers(t *testing.T) {
	server := fakeOpenRouter(t, `["10", "2", ""]`)
	defer server.Close()

	anime := &model.Anime{
		Seasons: []model.Season{
			{Number: 1, Episodes: []model.Episode{
				{Title: "Episódio 10", Number: "1"},
				{Title: "Episódio 2", Number: "2"},
				{Title: "Filme 1", Number: "3"},
			}},
		},
	}

	o := NewEpisodeOrganizer("test-key", "test-model", server.URL)
	changed, err := o.OrganizeAnime(context.Background(), anime)
	if err != nil {
		t.Fatalf("organize: %v", err)
	}
	if changed != 1 {
		t.Errorf("expected 1 changed, got %d", changed)
	}

	episodes := anime.Seasons[0].Episodes
	if episodes[0].Number != "10" {
		t.Errorf("expected episode 0 renumbered to 10, got %q", episodes[0].Number)
	}
	if episodes[1].Number != "2" {
		t.Errorf("expected episode 1 unchanged, got %q", episodes[1].Number)
	}
	if episodes[2].Number != "3" {
		t.Errorf("expected episode 2 unchanged on empty number, got %q", episodes[2].Number)
	}
}

func TestOrganizerDisabledWithoutKey(t *testing.T) {
	o := NewEpisodeOrganizer("", "", "")
	if o.Enabled() {
		t.Error("expected organizer disabled without api key")
	}
}
