package model

type UpscaleJob struct {
	ID               StringID `json:"id"`
	EpisodeID        StringID `json:"episodeId"`
	AnimeID          StringID `json:"animeId"`
	Type             string   `json:"type"`
	TargetHeight     int      `json:"targetHeight,omitempty"`
	BatchSize        *int     `json:"batchSize,omitempty"`
	Sharpen          *float64 `json:"sharpen,omitempty"`
	Saturation       *float64 `json:"saturation,omitempty"`
	Contrast         *float64 `json:"contrast,omitempty"`
	Interpolate      bool     `json:"interpolate,omitempty"`
	RunPodJobID      string   `json:"-"`
	SourceStorageKey string   `json:"-"`
	ResultStorageKey string   `json:"resultStorageKey,omitempty"`
	Status           string   `json:"status"`
	Error            string   `json:"error,omitempty"`
	AnimeTitle       string   `json:"animeTitle"`
	EpisodeTitle     string   `json:"episodeTitle"`
	EpisodeNumber    string   `json:"episodeNumber"`
	SeasonNumber     int      `json:"seasonNumber"`
	AnimeImageURL    string   `json:"animeImageUrl"`
	CreatedAt        string   `json:"createdAt,omitempty"`
	UpdatedAt        string   `json:"updatedAt,omitempty"`
}

type CreateUpscaleRequest struct {
	AnimeID      StringID   `json:"animeId"`
	EpisodeIDs   []StringID `json:"episodeIds"`
	TargetHeight int        `json:"targetHeight,omitempty"`
	BatchSize    *int       `json:"batchSize,omitempty"`
	Sharpen      *float64   `json:"sharpen,omitempty"`
	Saturation   *float64   `json:"saturation,omitempty"`
	Contrast     *float64   `json:"contrast,omitempty"`
	Interpolate  bool       `json:"interpolate,omitempty"`
}
