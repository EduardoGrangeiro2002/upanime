package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"
)

type GeoLookup interface {
	Lookup(ctx context.Context, ip string) string
}

type IPAPIGeo struct {
	client  *http.Client
	baseURL string
}

func NewIPAPIGeo() *IPAPIGeo {
	return &IPAPIGeo{
		client:  &http.Client{Timeout: 3 * time.Second},
		baseURL: "http://ip-api.com/json",
	}
}

func (g *IPAPIGeo) Lookup(ctx context.Context, ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsPrivate() || parsed.IsLoopback() {
		return ""
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/"+ip+"?fields=status,country,city", nil)
	if err != nil {
		return ""
	}
	response, err := g.client.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()

	var payload struct {
		Status  string `json:"status"`
		Country string `json:"country"`
		City    string `json:"city"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ""
	}
	if payload.Status != "success" {
		return ""
	}

	parts := []string{}
	if payload.City != "" {
		parts = append(parts, payload.City)
	}
	if payload.Country != "" {
		parts = append(parts, payload.Country)
	}
	return strings.Join(parts, ", ")
}
