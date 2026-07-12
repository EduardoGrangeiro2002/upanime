package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabasePath   string
	DownloadPath   string
	ScraperDir     string
	Port           string
	APIBaseURL     string
	RunPodEndpointID string
	RunPodAPIKey     string
	StorageType    string
	R2AccountID    string
	R2AccessKeyID  string
	R2AccessSecret string
	R2BucketName   string
	MaxDownloads   int
	OpenRouterAPIKey string
	ClassifierModel  string
	RedisAddr        string
	AuthSecret       string
	AuthCookieSecure bool
	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	SMTPFrom         string
}

func Load() *Config {
	godotenv.Load("../.env")

	return &Config{
		DatabasePath:   getEnv("DATABASE_PATH", "./upanime.db"),
		DownloadPath:   getEnv("DOWNLOAD_PATH", "./downloads"),
		ScraperDir:     getEnv("SCRAPER_DIR", "../scraper"),
		Port:           getEnv("PORT", "8080"),
		APIBaseURL:     getEnv("API_BASE_URL", "http://127.0.0.1:8080"),
		RunPodEndpointID: getEnv("RUNPOD_ENDPOINT_ID", ""),
		RunPodAPIKey:     getEnv("RUNPOD_API_KEY", ""),
		StorageType:    getEnv("STORAGE_TYPE", "local"),
		R2AccountID:    getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2AccessSecret: getEnv("R2_ACCESS_SECRET", ""),
		R2BucketName:   getEnv("R2_BUCKET_NAME", ""),
		MaxDownloads:   getEnvInt("MAX_DOWNLOADS", 3),
		OpenRouterAPIKey: getEnv("OPENROUTER_API_KEY", ""),
		ClassifierModel:  getEnv("CLASSIFIER_MODEL", ""),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		AuthSecret:       getEnv("AUTH_SECRET", ""),
		AuthCookieSecure: getEnv("AUTH_COOKIE_SECURE", "0") == "1",
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         getEnv("SMTP_PORT", "587"),
		SMTPUsername:     getEnv("SMTP_USERNAME", ""),
		SMTPPassword:     getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:         getEnv("SMTP_FROM", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
