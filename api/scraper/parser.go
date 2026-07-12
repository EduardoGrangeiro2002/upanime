package scraper

import (
	"encoding/json"
	"errors"
	"fmt"

	"upanime/api/model"
)

type ScrapeAnimeOutput struct {
	Title       string              `json:"title"`
	URL         string              `json:"url"`
	ImageURL    string              `json:"imageUrl"`
	Description string              `json:"description"`
	Episodes    []ScrapeEpisodeEntry `json:"episodes"`
}

type ScrapeEpisodeEntry struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Season string `json:"season"`
	Number string `json:"number"`
}

func (o *ScrapeAnimeOutput) Validate() error {
	if o.Title == "" {
		return errors.New("title is required")
	}
	if o.URL == "" {
		return errors.New("url is required")
	}
	if len(o.Episodes) == 0 {
		return errors.New("at least one episode is required")
	}
	return nil
}

func ParseScrapeOutput(data []byte) (*model.Anime, error) {
	var result ScrapeAnimeOutput
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse scrape output: %w", err)
	}

	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scrape output: %w", err)
	}

	seasonMap := make(map[string]*model.Season)
	var seasonOrder []string

	for _, ep := range result.Episodes {
		key := ep.Season
		if _, ok := seasonMap[key]; !ok {
			num := 1
			fmt.Sscanf(ep.Season, "%d", &num)
			seasonMap[key] = &model.Season{
				Number: num,
				Label:  fmt.Sprintf("Season %s", ep.Season),
				Type:   "episode",
			}
			seasonOrder = append(seasonOrder, key)
		}

		seasonMap[key].Episodes = append(seasonMap[key].Episodes, model.Episode{
			Title:  ep.Title,
			Number: ep.Number,
			URL:    ep.URL,
			Type:   "episode",
		})
	}

	var seasons []model.Season
	for _, key := range seasonOrder {
		seasons = append(seasons, *seasonMap[key])
	}

	return &model.Anime{
		Title:       result.Title,
		URL:         result.URL,
		ImageURL:    result.ImageURL,
		Description: result.Description,
		Seasons:     seasons,
	}, nil
}
