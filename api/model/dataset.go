package model

type DatasetSample struct {
	ID          StringID `json:"id"`
	Source      string   `json:"source"`
	Class       string   `json:"class"`
	FrameKey    string   `json:"-"`
	MaskKey     string   `json:"-"`
	FrameURL    string   `json:"frameUrl,omitempty"`
	MaskURL     string   `json:"maskUrl,omitempty"`
	AnimeTitle  string   `json:"animeTitle"`
	Episode     string   `json:"episode"`
	TimestampS  float64  `json:"timestampS"`
	TeacherProb float64  `json:"teacherProb"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"createdAt"`
	ReviewedAt  string   `json:"reviewedAt,omitempty"`
}

type DatasetClassStat struct {
	Class  string `json:"class"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type DatasetStats struct {
	Total     int                `json:"total"`
	Pending   int                `json:"pending"`
	Approved  int                `json:"approved"`
	Rejected  int                `json:"rejected"`
	NeedsEdit int                `json:"needsEdit"`
	ByClass   []DatasetClassStat `json:"byClass"`
}
