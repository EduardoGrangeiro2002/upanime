package model

type WatchProgress struct {
	EpisodeID     StringID `json:"episodeId"`
	AnimeID       StringID `json:"animeId,omitempty"`
	AnimeTitle    string   `json:"animeTitle,omitempty"`
	AnimeImageURL string   `json:"animeImageUrl,omitempty"`
	EpisodeTitle  string   `json:"episodeTitle,omitempty"`
	EpisodeNumber string   `json:"episodeNumber,omitempty"`
	SeasonNumber  int      `json:"seasonNumber,omitempty"`
	Position      float64  `json:"position"`
	Duration      float64  `json:"duration"`
	UpdatedAt     string   `json:"updatedAt"`
}
