package model

type Scraper struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	ScriptPath string `json:"scriptPath"`
	Active     bool   `json:"active"`
	CreatedAt  string `json:"createdAt"`
}
