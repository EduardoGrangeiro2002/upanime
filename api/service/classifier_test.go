package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"upanime/api/model"
	"upanime/api/store"
	"upanime/api/testutil"
)

func fakeOpenRouterServer(t *testing.T, responseText string, calls *atomic.Int64) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected authorization header %q", got)
		}
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Model != "anthropic/claude-sonnet-5" {
			t.Errorf("unexpected model %q", req.Model)
		}
		if len(req.Messages) != 2 || req.Messages[0].Role != "system" || req.Messages[1].Role != "user" {
			t.Errorf("unexpected messages %+v", req.Messages)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "gen-test",
			"model": "anthropic/claude-sonnet-5",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": responseText}},
			},
		})
	}))
	t.Cleanup(server.Close)
	return server
}

func seedAnime(t *testing.T, animes *store.SQLiteAnimeStore, title string, genres []string) *model.Anime {
	t.Helper()
	anime := &model.Anime{
		Title:       title,
		URL:         "https://animesonlinecc.to/anime/" + title,
		Description: "Uma descrição de teste.",
		Genres:      genres,
		ScraperID:   1,
	}
	if err := animes.Create(t.Context(), anime); err != nil {
		t.Fatalf("seed anime: %v", err)
	}
	return anime
}

func TestGenreClassifier_ClassifiesAndPersists(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	anime := seedAnime(t, animes, "classify-me", nil)

	var calls atomic.Int64
	server := fakeOpenRouterServer(t, `["Ação", "Comédia"]`, &calls)

	c := NewGenreClassifier("test-key", "", server.URL, animes)
	if !c.Enabled() {
		t.Fatal("expected classifier enabled with api key")
	}

	if err := c.ClassifyAndSave(t.Context(), anime.ID.Int64()); err != nil {
		t.Fatalf("classify: %v", err)
	}

	saved, err := animes.GetByID(t.Context(), anime.ID.Int64())
	if err != nil {
		t.Fatalf("load anime: %v", err)
	}
	if len(saved.Genres) != 2 || saved.Genres[0] != "Ação" || saved.Genres[1] != "Comédia" {
		t.Fatalf("expected genres persisted, got %v", saved.Genres)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 API call, got %d", calls.Load())
	}
}

func TestGenreClassifier_SkipsAlreadyClassified(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	anime := seedAnime(t, animes, "already-done", []string{"Drama"})

	var calls atomic.Int64
	server := fakeOpenRouterServer(t, `["Ação"]`, &calls)

	c := NewGenreClassifier("test-key", "", server.URL, animes)
	if err := c.ClassifyAndSave(t.Context(), anime.ID.Int64()); err != nil {
		t.Fatalf("classify: %v", err)
	}

	if calls.Load() != 0 {
		t.Fatalf("expected no API call for classified anime, got %d", calls.Load())
	}

	saved, _ := animes.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Genres) != 1 || saved.Genres[0] != "Drama" {
		t.Fatalf("expected genres untouched, got %v", saved.Genres)
	}
}

func TestGenreClassifier_InvalidResponseKeepsGenresEmpty(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	anime := seedAnime(t, animes, "bad-response", nil)

	var calls atomic.Int64
	server := fakeOpenRouterServer(t, "não sei classificar", &calls)

	c := NewGenreClassifier("test-key", "", server.URL, animes)
	if err := c.ClassifyAndSave(t.Context(), anime.ID.Int64()); err == nil {
		t.Fatal("expected error for unparseable response")
	}

	saved, _ := animes.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Genres) != 0 {
		t.Fatalf("expected genres to stay empty, got %v", saved.Genres)
	}
}

func TestGenreClassifier_APIErrorSurfaces(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	anime := seedAnime(t, animes, "api-error", nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"error":{"message":"insufficient credits"}}`))
	}))
	t.Cleanup(server.Close)

	c := NewGenreClassifier("test-key", "", server.URL, animes)
	err := c.ClassifyAndSave(t.Context(), anime.ID.Int64())
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}

	saved, _ := animes.GetByID(t.Context(), anime.ID.Int64())
	if len(saved.Genres) != 0 {
		t.Fatalf("expected genres to stay empty, got %v", saved.Genres)
	}
}

func TestGenreClassifier_ClassifyAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)
	first := seedAnime(t, animes, "bulk-one", nil)
	second := seedAnime(t, animes, "bulk-two", nil)
	seedAnime(t, animes, "bulk-done", []string{"Drama"})

	var calls atomic.Int64
	server := fakeOpenRouterServer(t, `["Ação"]`, &calls)

	c := NewGenreClassifier("test-key", "", server.URL, animes)
	result, err := c.ClassifyAll(t.Context())
	if err != nil {
		t.Fatalf("classify all: %v", err)
	}

	if len(result.Classified) != 2 {
		t.Fatalf("expected 2 classified, got %+v", result.Classified)
	}
	if result.Skipped != 1 {
		t.Fatalf("expected 1 skipped, got %d", result.Skipped)
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected no failures, got %+v", result.Failed)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 API calls, got %d", calls.Load())
	}

	for _, id := range []int64{first.ID.Int64(), second.ID.Int64()} {
		saved, err := animes.GetByID(t.Context(), id)
		if err != nil {
			t.Fatalf("load anime %d: %v", id, err)
		}
		if len(saved.Genres) != 1 || saved.Genres[0] != "Ação" {
			t.Fatalf("expected genres persisted for %d, got %v", id, saved.Genres)
		}
	}
}

func TestGenreClassifier_ClassifyAllDisabled(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)

	c := NewGenreClassifier("", "", "", animes)
	if _, err := c.ClassifyAll(t.Context()); err == nil {
		t.Fatal("expected error when classifier disabled")
	}
}

func TestGenreClassifier_DisabledWithoutKey(t *testing.T) {
	db := testutil.NewTestDB(t)
	animes := store.NewSQLiteAnimeStore(db)

	c := NewGenreClassifier("", "", "", animes)
	if c.Enabled() {
		t.Fatal("expected classifier disabled without api key")
	}
	c.ClassifyAsync(1)
}

func TestParseGenres(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"plain array", `["Ação", "Romance"]`, []string{"Ação", "Romance"}, false},
		{"array with prose around", "Claro! Aqui está: [\"Terror\"] espero que ajude", []string{"Terror"}, false},
		{"normalizes case and dedupes", `["ação", "AÇÃO", "drama"]`, []string{"Ação", "Drama"}, false},
		{"caps at three", `["Ação", "Drama", "Comédia", "Romance"]`, []string{"Ação", "Drama", "Comédia"}, false},
		{"drops unknown genres", `["Ação", "Culinária"]`, []string{"Ação"}, false},
		{"all unknown", `["Culinária"]`, nil, true},
		{"no array", "sem json aqui", nil, true},
		{"invalid json", "[não é json]", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseGenres(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("expected %v, got %v", tc.want, got)
				}
			}
		})
	}
}
