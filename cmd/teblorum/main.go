package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/teblorum/teblorum/internal/config"
	"github.com/teblorum/teblorum/internal/handler"
	"github.com/teblorum/teblorum/internal/render"
	"github.com/teblorum/teblorum/internal/repo"
	"github.com/teblorum/teblorum/internal/service"
	"github.com/teblorum/teblorum/webassets"
)

func main() {
	// --- Конфигурация ---
	cfgPath := os.Getenv("TEBLORUM_CONFIG")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// --- База данных ---
	db, err := repo.OpenDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	// --- Bootstrap root-пользователя ---
	if cfg.BootstrapEmail != "" {
		if err := config.BootstrapRoot(db, cfg.BootstrapEmail, cfg.BootstrapPassword); err != nil {
			log.Printf("bootstrap root: %v", err)
		}
	}

	// --- Рендерер ---
	renderer, err := render.NewTemplateRenderer(webassets.FS)
	if err != nil {
		log.Fatalf("init renderer: %v", err)
	}

	// --- Сервисы ---
	userSvc := service.NewUserService(db)
	postSvc := service.NewPostService(db)
	commentSvc := service.NewCommentService(db)
	messageSvc := service.NewMessageService(db)
	modSvc := service.NewModerationService(db)
	rootSvc := service.NewRootService(db, cfg.BackupDir)

	deps := &handler.Dependencies{
		DB:       db,
		Renderer: renderer,
		Users:    userSvc,
		Posts:    postSvc,
		Comments: commentSvc,
		Messages: messageSvc,
		Mod:      modSvc,
		Root:     rootSvc,

		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		GoogleRedirectURL:  cfg.GoogleRedirectURL,
		SessionTTL:         cfg.SessionTTLDays,
	}

	// --- Статика ---
	staticFS, err := fs.Sub(webassets.FS, "web/static")
	if err != nil {
		log.Fatalf("open static fs: %v", err)
	}

	// --- Роутер с полной middleware-цепочкой ---
	h := handler.SetupRoutes(deps, staticFS)

	// --- Фоновая очистка ---
	cleanup := service.NewCleanupService(db, 1*time.Hour)
	cleanup.Start()
	defer cleanup.Stop()

	// --- Запуск (всегда HTTP; для production используйте reverse proxy) ---
	addr := cfg.Addr
	log.Printf("teblorum запущен на %s", addr)
	log.Printf("режим: HTTP (для production настройте reverse proxy — см. deploy/PRODUCTION.md)")

	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
