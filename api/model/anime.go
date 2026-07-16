package model

import (
	"encoding/json"
	"strconv"
)

type StringID int64

func (s StringID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(s), 10))
}

func (s *StringID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		var num int64
		if err := json.Unmarshal(data, &num); err != nil {
			return err
		}
		*s = StringID(num)
		return nil
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}
	*s = StringID(n)
	return nil
}

func (s StringID) Int64() int64 {
	return int64(s)
}

type Anime struct {
	ID          StringID `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	ImageURL    string   `json:"imageUrl"`
	Description string   `json:"description"`
	CoverPath   string   `json:"coverPath,omitempty"`
	CoverURL    string   `json:"coverUrl,omitempty"`
	Genres      []string `json:"genres"`
	ScraperID   int64    `json:"-"`
	Seasons     []Season `json:"seasons"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`
}

type Season struct {
	ID       int64     `json:"-"`
	AnimeID  int64     `json:"-"`
	Number   int       `json:"number"`
	Label    string    `json:"label"`
	Type     string    `json:"type"`
	Episodes []Episode `json:"episodes"`
}

type Episode struct {
	ID                 StringID         `json:"id"`
	SeasonID           int64            `json:"-"`
	AnimeID            int64            `json:"-"`
	Title              string           `json:"title"`
	Number             string           `json:"number"`
	SeasonNumber       int              `json:"seasonNumber"`
	URL                string           `json:"url"`
	Type               string           `json:"type"`
	StorageKey         string           `json:"storageKey,omitempty"`
	UpscaledStorageKey string           `json:"upscaledStorageKey,omitempty"`
	UpscaledVariants   []EpisodeVariant `json:"upscaledVariants,omitempty"`
}
