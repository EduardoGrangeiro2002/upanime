package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"upanime/api/model"
)

const organizerSystemPrompt = `Você organiza episódios de anime. Receberá uma lista numerada de títulos de episódios. Responda APENAS com um array JSON de strings com o MESMO tamanho da lista, onde cada posição contém o número do episódio extraído do título correspondente, sem zeros à esquerda (ex.: "1", "26", "153"). Use "" quando o título não indicar um número de episódio (filme, OVA, especial). Não escreva nada além do array JSON.`

type EpisodeOrganizer struct {
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
	enabled bool
}

func NewEpisodeOrganizer(apiKey, model, baseURL string) *EpisodeOrganizer {
	if model == "" {
		model = DefaultClassifierModel
	}
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	if apiKey == "" {
		log.Println("episode organizer disabled: OPENROUTER_API_KEY not set")
		return &EpisodeOrganizer{model: model, baseURL: baseURL}
	}
	return &EpisodeOrganizer{
		client:  &http.Client{Timeout: 120 * time.Second},
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		enabled: true,
	}
}

func (o *EpisodeOrganizer) Enabled() bool {
	return o != nil && o.enabled
}

func (o *EpisodeOrganizer) Renumber(ctx context.Context, titles []string) ([]string, error) {
	if len(titles) == 0 {
		return []string{}, nil
	}

	var prompt strings.Builder
	for i, title := range titles {
		fmt.Fprintf(&prompt, "%d. %s\n", i+1, title)
	}

	maxTokens := 1024 + 8*len(titles)
	text, err := chatComplete(ctx, o.client, o.baseURL, o.apiKey, o.model, organizerSystemPrompt, prompt.String(), maxTokens)
	if err != nil {
		return nil, err
	}

	numbers, err := parseNumbers(text, len(titles))
	if err != nil {
		return nil, fmt.Errorf("parse response %q: %w", text, err)
	}
	return numbers, nil
}

func (o *EpisodeOrganizer) OrganizeAnime(ctx context.Context, anime *model.Anime) (int, error) {
	var episodes []*model.Episode
	for s := range anime.Seasons {
		for e := range anime.Seasons[s].Episodes {
			episodes = append(episodes, &anime.Seasons[s].Episodes[e])
		}
	}
	if len(episodes) == 0 {
		return 0, nil
	}

	titles := make([]string, len(episodes))
	for i, ep := range episodes {
		titles[i] = ep.Title
	}

	numbers, err := o.Renumber(ctx, titles)
	if err != nil {
		return 0, err
	}

	changed := 0
	for i, ep := range episodes {
		if numbers[i] == "" || numbers[i] == ep.Number {
			continue
		}
		ep.Number = numbers[i]
		changed++
	}
	return changed, nil
}

func parseNumbers(text string, want int) ([]string, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array found")
	}

	var raw []string
	if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}
	if len(raw) != want {
		return nil, fmt.Errorf("expected %d numbers, got %d", want, len(raw))
	}

	numbers := make([]string, len(raw))
	for i, n := range raw {
		numbers[i] = normalizeEpisodeNumber(n)
	}
	return numbers, nil
}

func normalizeEpisodeNumber(n string) string {
	n = strings.TrimSpace(n)
	if n == "" {
		return ""
	}
	for _, r := range n {
		if r < '0' || r > '9' {
			return ""
		}
	}
	trimmed := strings.TrimLeft(n, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}
