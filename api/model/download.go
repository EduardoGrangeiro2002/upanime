package model

type Download struct {
	ID            StringID `json:"id"`
	EpisodeID     StringID `json:"episodeId"`
	AnimeID       StringID `json:"animeId"`
	Status        string   `json:"status"`
	Progress      int      `json:"progress"`
	Speed         string   `json:"speed"`
	ETA           string   `json:"eta"`
	Error         string   `json:"error,omitempty"`
	DestPath      string   `json:"-"`
	EpisodeTitle  string   `json:"episodeTitle"`
	EpisodeNumber string   `json:"episodeNumber"`
	SeasonNumber  int      `json:"seasonNumber"`
	AnimeTitle    string   `json:"animeTitle"`
	AnimeImageURL string   `json:"animeImageUrl"`
	CreatedAt     string   `json:"createdAt,omitempty"`
	UpdatedAt     string   `json:"updatedAt,omitempty"`
}

type DownloadEpisodeInput struct {
	Title        string `json:"title"`
	Number       string `json:"number"`
	URL          string `json:"url"`
	SeasonNumber int    `json:"seasonNumber"`
}

type CreateDownloadsRequest struct {
	AnimeID       StringID               `json:"animeId"`
	AnimeTitle    string                 `json:"animeTitle"`
	AnimeImageURL string                 `json:"animeImageUrl"`
	Description   string                 `json:"description"`
	SourceURL     string                 `json:"sourceUrl"`
	SeasonNumber  int                    `json:"seasonNumber"`
	Episodes      []DownloadEpisodeInput `json:"episodes"`
}
