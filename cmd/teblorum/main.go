package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/teblorum/teblorum/internal/render"
	"github.com/teblorum/teblorum/internal/repo"
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
	_ = renderer

	// --- Роутер ---
	mux := http.NewServeMux()

	// Статика
	subFS, err := fs.Sub(webassets.FS, "web/static")
	if err != nil {
		log.Fatalf("open static fs: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	// --- Заглушки маршрутов (будут реализованы в Этапе 5) ---
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		renderer.PageHTTP(w, "feed", render.FeedPageData{
			PageData: render.PageData{Title: "Лента", Theme: "light"},
		}, render.DetectHTMX(r))
	})

	// --- Запуск ---
	addr := os.Getenv("TEBLORUM_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("teblorum запущен на %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
