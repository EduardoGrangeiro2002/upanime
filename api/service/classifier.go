package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"upanime/api/store"
)

const DefaultClassifierModel = "anthropic/claude-sonnet-5"

const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

var GenreTaxonomy = []string{
	"Ação",
	"Aventura",
	"Comédia",
	"Drama",
	"Esporte",
	"Fantasia",
	"Ficção Científica",
	"Histórico",
	"Mecha",
	"Mistério",
	"Musical",
	"Romance",
	"Slice of Life",
	"Sobrenatural",
	"Suspense",
	"Terror",
}

const classifierSystemPrompt = `Você é um classificador de gêneros de anime. Dado o título e a descrição de um anime, responda APENAS com um array JSON contendo de 1 a 3 gêneros, escolhidos EXATAMENTE desta lista: %s. Não escreva nada além do array JSON.`

type GenreClassifier struct {
	animes  store.AnimeStore
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
	enabled bool
}

func NewGenreClassifier(apiKey, model, baseURL string, animes store.AnimeStore) *GenreClassifier {
	if model == "" {
		model = DefaultClassifierModel
	}
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	if apiKey == "" {
		log.Println("genre classifier disabled: OPENROUTER_API_KEY not set")
		return &GenreClassifier{animes: animes, model: model, baseURL: baseURL}
	}
	return &GenreClassifier{
		animes:  animes,
		client:  &http.Client{Timeout: 60 * time.Second},
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		enabled: true,
	}
}

func (c *GenreClassifier) Enabled() bool {
	return c != nil && c.enabled
}

func (c *GenreClassifier) ClassifyAsync(animeID int64) {
	if !c.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := c.ClassifyAndSave(ctx, animeID); err != nil {
			log.Printf("genre classification failed for anime %d: %v", animeID, err)
		}
	}()
}

func (c *GenreClassifier) ClassifyAndSave(ctx context.Context, animeID int64) error {
	anime, err := c.animes.GetByID(ctx, animeID)
	if err != nil {
		return fmt.Errorf("load anime: %w", err)
	}
	if len(anime.Genres) > 0 {
		return nil
	}

	genres, err := c.Classify(ctx, anime.Title, anime.Description)
	if err != nil {
		return err
	}

	if err := c.animes.UpdateGenres(ctx, animeID, genres); err != nil {
		return fmt.Errorf("save genres: %w", err)
	}
	log.Printf("anime %d (%s) classified as %v", animeID, anime.Title, genres)
	return nil
}

type ClassifiedAnime struct {
	ID     int64    `json:"id,string"`
	Title  string   `json:"title"`
	Genres []string `json:"genres"`
}

type FailedAnime struct {
	ID    int64  `json:"id,string"`
	Title string `json:"title"`
	Error string `json:"error"`
}

type ClassifyAllResult struct {
	Classified []ClassifiedAnime `json:"classified"`
	Skipped    int               `json:"skipped"`
	Failed     []FailedAnime     `json:"failed"`
}

func (c *GenreClassifier) ClassifyAll(ctx context.Context) (*ClassifyAllResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("classifier disabled: OPENROUTER_API_KEY not set")
	}

	animes, err := c.animes.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list animes: %w", err)
	}

	result := &ClassifyAllResult{Classified: []ClassifiedAnime{}, Failed: []FailedAnime{}}
	for _, anime := range animes {
		if len(anime.Genres) > 0 {
			result.Skipped++
			continue
		}

		genres, err := c.Classify(ctx, anime.Title, anime.Description)
		if err != nil {
			log.Printf("classify all: anime %d (%s) failed: %v", anime.ID.Int64(), anime.Title, err)
			result.Failed = append(result.Failed, FailedAnime{ID: anime.ID.Int64(), Title: anime.Title, Error: err.Error()})
			continue
		}

		if err := c.animes.UpdateGenres(ctx, anime.ID.Int64(), genres); err != nil {
			result.Failed = append(result.Failed, FailedAnime{ID: anime.ID.Int64(), Title: anime.Title, Error: err.Error()})
			continue
		}

		log.Printf("classify all: anime %d (%s) → %v", anime.ID.Int64(), anime.Title, genres)
		result.Classified = append(result.Classified, ClassifiedAnime{ID: anime.ID.Int64(), Title: anime.Title, Genres: genres})
	}

	return result, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *GenreClassifier) Classify(ctx context.Context, title, description string) ([]string, error) {
	taxonomy, _ := json.Marshal(GenreTaxonomy)
	system := fmt.Sprintf(classifierSystemPrompt, string(taxonomy))
	prompt := fmt.Sprintf("Título: %s\nDescrição: %s", title, description)

	text, err := chatComplete(ctx, c.client, c.baseURL, c.apiKey, c.model, system, prompt, 256)
	if err != nil {
		return nil, err
	}

	genres, err := parseGenres(text)
	if err != nil {
		return nil, fmt.Errorf("parse response %q: %w", text, err)
	}
	return genres, nil
}

func chatComplete(ctx context.Context, client *http.Client, baseURL, apiKey, model, system, prompt string, maxTokens int) (string, error) {
	body, err := json.Marshal(chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
		MaxTokens: maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "upanime")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openrouter status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("openrouter error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty choices in response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func parseGenres(text string) ([]string, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array found")
	}

	var raw []string
	if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}

	canonical := make(map[string]string, len(GenreTaxonomy))
	for _, g := range GenreTaxonomy {
		canonical[strings.ToLower(g)] = g
	}

	var genres []string
	seen := make(map[string]bool)
	for _, g := range raw {
		match, ok := canonical[strings.ToLower(strings.TrimSpace(g))]
		if !ok || seen[match] {
			continue
		}
		seen[match] = true
		genres = append(genres, match)
		if len(genres) == 3 {
			break
		}
	}

	if len(genres) == 0 {
		return nil, fmt.Errorf("no valid genres in %v", raw)
	}
	return genres, nil
}
