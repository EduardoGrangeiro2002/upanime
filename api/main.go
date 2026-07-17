package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"upanime/api/auth"
	"upanime/api/config"
	"upanime/api/db"
	"upanime/api/handler"
	"upanime/api/model"
	"upanime/api/scraper"
	"upanime/api/service"
	"upanime/api/storage"
	"upanime/api/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "create-user" {
		if err := runCreateUser(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "classify" {
		if err := runClassify(); err != nil {
			log.Fatal(err)
		}
		return
	}

	cfg := config.Load()

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatal(err)
	}

	animeStore := store.NewSQLiteAnimeStore(database)
	downloadStore := store.NewSQLiteDownloadStore(database)
	episodeStore := store.NewSQLiteEpisodeStore(database)
	scraperStore := store.NewSQLiteScraperStore(database)

	var fs storage.FileStorage
	if cfg.StorageType == "r2" {
		fs = storage.NewR2Storage(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2AccessSecret, cfg.R2BucketName)
	} else {
		fs = storage.NewLocalStorage(cfg.DownloadPath)
	}

	exec := scraper.NewPythonExecutor(cfg.ScraperDir)

	classifier := service.NewGenreClassifier(cfg.OpenRouterAPIKey, cfg.ClassifierModel, "", animeStore)
	organizer := service.NewEpisodeOrganizer(cfg.OpenRouterAPIKey, cfg.ClassifierModel, "")

	animeHandler := handler.NewAnimeHandler(scraperStore, exec, organizer)
	downloadHandler := handler.NewDownloadHandler(downloadStore, animeStore, episodeStore, scraperStore, exec, fs, classifier, cfg.DatabasePath, cfg.MaxDownloads)
	catalogHandler := handler.NewCatalogHandler(animeStore, episodeStore, fs)
	uploadHandler := handler.NewUploadHandler(animeStore, episodeStore, scraperStore, fs, classifier)

	upscaleStore := store.NewSQLiteUpscaleStore(database)
	workerClient := service.NewRunPodUpscaleWorkerClient(cfg.RunPodEndpointID, cfg.RunPodAPIKey)
	editionHandler := handler.NewEditionHandler(upscaleStore, animeStore, episodeStore, fs, workerClient)

	thumbnailService := service.NewThumbnailService(fs, nil)
	thumbnailHandler := handler.NewThumbnailHandler(episodeStore, thumbnailService, fs)

	mlDatasetPath := cfg.MLDatasetPath
	if mlDatasetPath == "" {
		mlDatasetPath = filepath.Join(filepath.Dir(cfg.DatabasePath), "ml_dataset.db")
	}
	mlDatabase, err := db.Open(mlDatasetPath)
	if err != nil {
		log.Fatal(err)
	}
	defer mlDatabase.Close()
	datasetStore, err := store.NewSQLiteDatasetStore(mlDatabase)
	if err != nil {
		log.Fatal(err)
	}
	datasetHandler := handler.NewDatasetHandler(datasetStore, fs)

	poller := service.NewRunPodPoller(upscaleStore, episodeStore, workerClient, 10*time.Second)
	poller.Start()
	defer poller.Stop()

	authSecret := cfg.AuthSecret
	if authSecret == "" {
		generated, err := auth.GenerateSecret()
		if err != nil {
			log.Fatal(err)
		}
		authSecret = generated
		log.Println("AUTH_SECRET não definido — usando segredo aleatório; sessões serão invalidadas a cada restart")
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("aviso: redis inacessível em %s (%v) — MFA e reset de senha não vão funcionar", cfg.RedisAddr, err)
	}

	var mailer auth.Mailer = &auth.LogMailer{}
	if cfg.SMTPHost != "" {
		mailer = auth.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	}

	userStore := store.NewSQLiteUserStore(database)
	codeStore := auth.NewCodeStore(redisClient)
	authService := auth.NewService(
		userStore,
		codeStore,
		mailer,
		auth.NewIPAPIGeo(),
		auth.NewTokenSigner(authSecret),
		time.Now,
	)
	authHandler := handler.NewAuthHandler(authService, userStore, cfg.AuthCookieSecure)
	inviteHandler := handler.NewInviteHandler(userStore, mailer)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/health", handler.HealthHandler())

	r.Route("/api/auth", func(ar chi.Router) {
		ar.Use(handler.RateLimitAuth(codeStore))
		ar.Post("/login", authHandler.Login)
		ar.Post("/change-password", authHandler.ChangePassword)
		ar.Post("/mfa", authHandler.VerifyMFA)
		ar.Post("/forgot", authHandler.Forgot)
		ar.Post("/reset", authHandler.Reset)
		ar.Post("/logout", authHandler.Logout)
	})

	r.Group(func(pr chi.Router) {
		pr.Use(handler.RequireAuth(authService))

		pr.Get("/api/auth/me", authHandler.Me)
		pr.Get("/api/anime", animeHandler.Get)
		pr.Post("/api/downloads", downloadHandler.Create)
		pr.Get("/api/downloads", downloadHandler.List)
		pr.Get("/api/downloads/{id}", downloadHandler.GetByID)
		pr.Delete("/api/downloads/{id}", downloadHandler.Delete)
		pr.Get("/api/catalog", catalogHandler.List)
		pr.Post("/api/catalog/upload", uploadHandler.Create)
		pr.Post("/api/catalog/classify", handler.ClassifyAllHandler(classifier))
		pr.Post("/api/catalog/anime/{id}/organize", handler.OrganizeAnimeHandler(organizer, animeStore))
		pr.Delete("/api/catalog/anime/{id}", catalogHandler.DeleteAnime)
		pr.Post("/api/catalog/anime/{id}/cover", catalogHandler.UploadCover)
		pr.Delete("/api/catalog/episode/{id}", catalogHandler.DeleteEpisode)
		pr.Delete("/api/catalog/episode/{id}/upscaled", catalogHandler.DeleteUpscaledEpisode)
		pr.Get("/api/catalog/episode/{id}/stream", catalogHandler.StreamURL)
		pr.Get("/api/catalog/episode/{id}/stream/file", catalogHandler.StreamFile)
		pr.Get("/api/catalog/episode/{id}/thumbnail", thumbnailHandler.Get)

		pr.Post("/api/upscale", editionHandler.Create)
		pr.Get("/api/upscale", editionHandler.List)
		pr.Delete("/api/upscale/{id}", editionHandler.Delete)

		pr.Post("/api/dataset/samples", datasetHandler.Ingest)
		pr.Get("/api/dataset/samples/queue", datasetHandler.Queue)
		pr.Post("/api/dataset/samples/{id}/verdict", datasetHandler.Verdict)
		pr.Get("/api/dataset/stats", datasetHandler.Stats)

		pr.Group(func(admin chi.Router) {
			admin.Use(handler.RequireAdmin(userStore))
			admin.Post("/api/invites", inviteHandler.Create)
			admin.Get("/api/users", inviteHandler.ListUsers)
		})
	})

	distPath := filepath.Join("..", "client", "dist")
	if _, err := os.Stat(distPath); err == nil {
		fileServer := http.FileServer(http.Dir(distPath))
		r.Handle("/*", fileServer)
	}

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

func runCreateUser(args []string) error {
	if len(args) != 1 || !strings.Contains(args[0], "@") {
		return errors.New("uso: upanime-api create-user <email>")
	}
	email := args[0]

	cfg := config.Load()
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		return err
	}

	password, err := auth.GenerateTempPassword()
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	user := &model.User{Email: email, PasswordHash: hash, MustChangePassword: true, IsAdmin: true}
	if err := store.NewSQLiteUserStore(database).Create(context.Background(), user); err != nil {
		return fmt.Errorf("criar usuário: %w", err)
	}

	fmt.Printf("Usuário admin criado com sucesso.\n  email: %s\n  senha temporária: %s\n\nNo primeiro login será exigida a troca da senha.\n", email, password)
	return nil
}

func runClassify() error {
	cfg := config.Load()
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		return err
	}

	classifier := service.NewGenreClassifier(cfg.OpenRouterAPIKey, cfg.ClassifierModel, "", store.NewSQLiteAnimeStore(database))
	if !classifier.Enabled() {
		return errors.New("classificador desabilitado: defina OPENROUTER_API_KEY")
	}

	fmt.Println("Classificando animes sem gênero via OpenRouter...")
	result, err := classifier.ClassifyAll(context.Background())
	if err != nil {
		return fmt.Errorf("classificar: %w", err)
	}

	for _, anime := range result.Classified {
		fmt.Printf("  ✓ %s → %v\n", anime.Title, anime.Genres)
	}
	for _, anime := range result.Failed {
		fmt.Printf("  ✗ %s: %s\n", anime.Title, anime.Error)
	}
	fmt.Printf("\nConcluído: %d classificados, %d já tinham gênero, %d falharam.\n", len(result.Classified), result.Skipped, len(result.Failed))
	return nil
}
