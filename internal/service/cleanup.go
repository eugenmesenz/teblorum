package service

import (
	"database/sql"
	"log"
	"time"

	"github.com/teblorum/teblorum/internal/repo"
)

// CleanupService выполняет фоновую очистку устаревших данных.
type CleanupService struct {
	db     *sql.DB
	ticker *time.Ticker
	stopCh chan struct{}
}

// NewCleanupService создаёт сервис фоновой очистки.
// interval — периодичность проверки (например, 1 час).
func NewCleanupService(db *sql.DB, interval time.Duration) *CleanupService {
	return &CleanupService{
		db:     db,
		ticker: time.NewTicker(interval),
		stopCh: make(chan struct{}),
	}
}

// Start запускает фоновую очистку в отдельной горутине.
func (s *CleanupService) Start() {
	go func() {
		log.Println("cleanup service started")
		for {
			select {
			case <-s.ticker.C:
				s.run()
			case <-s.stopCh:
				s.ticker.Stop()
				log.Println("cleanup service stopped")
				return
			}
		}
	}()
}

// Stop останавливает фоновую очистку.
func (s *CleanupService) Stop() {
	close(s.stopCh)
}

// RunOnce выполняет одноразовую очистку (для тестов или ручного запуска).
func (s *CleanupService) RunOnce() {
	s.run()
}

func (s *CleanupService) run() {
	sessions := repo.NewSessionRepo(s.db)

	// Очистка просроченных сессий
	if err := sessions.CleanExpired(); err != nil {
		log.Printf("cleanup sessions: %v", err)
	} else {
		log.Println("cleanup: expired sessions removed")
	}

	// Очистка просроченных токенов сброса пароля
	_, err := s.db.Exec("DELETE FROM password_resets WHERE expires_at < datetime('now')")
	if err != nil {
		log.Printf("cleanup password resets: %v", err)
	}

	// Снятие банов, срок которых истёк
	_, err = s.db.Exec(`
		UPDATE users SET banned_until = NULL, banned_by = NULL
		WHERE banned_until IS NOT NULL AND banned_until < datetime('now')
	`)
	if err != nil {
		log.Printf("cleanup expired bans: %v", err)
	}
}
