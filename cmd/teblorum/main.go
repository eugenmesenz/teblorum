package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/teblorum/teblorum/internal/handler"
	"github.com/teblorum/teblorum/internal/render"
	"github.com/teblorum/teblorum/internal/repo"
	"github.com/teblorum/teblorum/internal/service"
	"github.com/teblorum/teblorum/webassets"
)

func main() {
	// --- База данных ---
	dbPath := os.Getenv("TEBLORUM_DB_PATH")
	if dbPath == "" {
		dbPath = "teblorum.db"
	}

	db, err := repo.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

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
	rootSvc := service.NewRootService(db, "./backups")

	deps := &handler.Dependencies{
		DB:       db,
		Renderer: renderer,
		Users:    userSvc,
		Posts:    postSvc,
		Comments: commentSvc,
		Messages: messageSvc,
		Mod:      modSvc,
		Root:     rootSvc,

		GoogleClientID:     os.Getenv("TEBLORUM_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("TEBLORUM_GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("TEBLORUM_GOOGLE_REDIRECT_URL"),
		SessionTTL:         30,
	}

	// --- Статика ---
	staticFS, err := fs.Sub(webassets.FS, "web/static")
	if err != nil {
		log.Fatalf("open static fs: %v", err)
	}

	// --- Роутер с полной middleware-цепочкой ---
	h := handler.SetupRoutes(deps, staticFS)

	// --- Запуск ---
	addr := os.Getenv("TEBLORUM_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("teblorum запущен на %s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
