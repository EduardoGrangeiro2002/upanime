package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"upanime/api/handler"
	"upanime/api/jobs"
	"upanime/api/model"
	"upanime/api/service"
	"upanime/api/store"
	"upanime/api/testutil"
)

func fakeOpenRouterServer(t *testing.T, reply string) *httptest.Server {
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

func organizeRequest(h http.HandlerFunc, animeID string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.Post("/api/catalog/anime/{id}/organize", h)
	req := httptest.NewRequest("POST", "/api/catalog/anime/"+animeID+"/organize", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func seedOrganizeAnime(t *testing.T, animeStore *store.SQLiteAnimeStore) *model.Anime {
	t.Helper()
	anime := &model.Anime{
		Title:     "Organize Test",
		URL:       "https://animesonlinecc.to/anime/organize-test",
		ScraperID: 1,
		Seasons: []model.Season{
			{Number: 1, Label: "Season 1", Type: "episode", Episodes: []model.Episode{
				{Title: "Episódio 10 – Final", Number: "1", URL: "https://example.com/e10", Type: "episode"},
				{Title: "Episódio 1 – Início", Number: "2", URL: "https://example.com/e1", Type: "episode"},
			}},
		},
	}
	if err := animeStore.Create(t.Context(), anime); err != nil {
		t.Fatalf("create anime: %v", err)
	}
	return anime
}

func TestOrganizeAnimeHandler_EnqueuesAndAccepts(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	anime := seedOrganizeAnime(t, animeStore)

	server := fakeOpenRouterServer(t, `[]`)
	defer server.Close()
	organizer := service.NewEpisodeOrganizer("test-key", "test-model", server.URL)
	enq := &fakeEnqueuer{}

	w := organizeRequest(handler.OrganizeAnimeHandler(organizer, animeStore, enq), strconv.FormatInt(anime.ID.Int64(), 10))

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if len(enq.organized) != 1 || enq.organized[0] != anime.ID.Int64() {
		t.Fatalf("expected organize enqueued for %d, got %v", anime.ID.Int64(), enq.organized)
	}
}

func TestOrganizeTask_RenumbersAndReorders(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	anime := seedOrganizeAnime(t, animeStore)

	server := fakeOpenRouterServer(t, `["10", "1"]`)
	defer server.Close()
	organizer := service.NewEpisodeOrganizer("test-key", "test-model", server.URL)

	task := jobs.NewOrganizeTask(anime.ID.Int64())
	if err := handler.OrganizeTask(organizer, animeStore)(t.Context(), task); err != nil {
		t.Fatalf("organize task: %v", err)
	}

	updated, err := animeStore.GetByID(t.Context(), anime.ID.Int64())
	if err != nil {
		t.Fatalf("reload anime: %v", err)
	}

	episodes := updated.Seasons[0].Episodes
	if episodes[0].Title != "Episódio 1 – Início" || episodes[0].Number != "1" {
		t.Errorf("expected episode 1 first, got %q number %q", episodes[0].Title, episodes[0].Number)
	}
	if episodes[1].Title != "Episódio 10 – Final" || episodes[1].Number != "10" {
		t.Errorf("expected episode 10 last, got %q number %q", episodes[1].Title, episodes[1].Number)
	}
}

func TestOrganizeTask_DeletedAnimeIsNoop(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)

	server := fakeOpenRouterServer(t, `[]`)
	defer server.Close()
	organizer := service.NewEpisodeOrganizer("test-key", "test-model", server.URL)

	task := jobs.NewOrganizeTask(9999)
	if err := handler.OrganizeTask(organizer, animeStore)(t.Context(), task); err != nil {
		t.Fatalf("expected nil for missing anime, got %v", err)
	}
}

func TestOrganizeAnimeHandler_DisabledReturns503(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	organizer := service.NewEpisodeOrganizer("", "", "")

	w := organizeRequest(handler.OrganizeAnimeHandler(organizer, animeStore, &fakeEnqueuer{}), "1")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestOrganizeAnimeHandler_UnknownAnime(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)

	server := fakeOpenRouterServer(t, `[]`)
	defer server.Close()
	organizer := service.NewEpisodeOrganizer("test-key", "test-model", server.URL)
	enq := &fakeEnqueuer{}

	w := organizeRequest(handler.OrganizeAnimeHandler(organizer, animeStore, enq), "9999")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if len(enq.organized) != 0 {
		t.Fatal("expected nothing enqueued for unknown anime")
	}
}

func TestClassifyAllHandler_EnqueuesAndAccepts(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	classifier := service.NewGenreClassifier("test-key", "test-model", "", animeStore)
	enq := &fakeEnqueuer{}

	req := httptest.NewRequest("POST", "/api/catalog/classify", nil)
	w := httptest.NewRecorder()
	handler.ClassifyAllHandler(classifier, enq)(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if enq.classifyAlls != 1 {
		t.Fatalf("expected classify enqueued once, got %d", enq.classifyAlls)
	}
}

func TestClassifyAllHandler_DisabledReturns503(t *testing.T) {
	db := testutil.NewTestDB(t)
	animeStore := store.NewSQLiteAnimeStore(db)
	classifier := service.NewGenreClassifier("", "", "", animeStore)

	req := httptest.NewRequest("POST", "/api/catalog/classify", nil)
	w := httptest.NewRecorder()
	handler.ClassifyAllHandler(classifier, &fakeEnqueuer{})(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
