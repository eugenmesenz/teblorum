package service

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RootService — сервис для root-операций (бекап, восстановление).
type RootService struct {
	db        *sql.DB
	backupDir string
}

func NewRootService(db *sql.DB, backupDir string) *RootService {
	return &RootService{
		db:        db,
		backupDir: backupDir,
	}
}

// CreateBackup создаёт бекап базы данных через VACUUM INTO.
func (s *RootService) CreateBackup() (string, error) {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	filename := fmt.Sprintf("teblorum_backup_%s.db", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(s.backupDir, filename)

	// VACUUM INTO создаёт копию БД
	_, err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return "", fmt.Errorf("vacuum into: %w", err)
	}

	return backupPath, nil
}

// RestoreFromUpload восстанавливает базу данных из загруженного файла.
func (s *RootService) RestoreFromUpload(reader io.Reader, dbPath string) error {
	// Создаём временный файл
	tmpFile, err := os.CreateTemp(filepath.Dir(dbPath), "teblorum_restore_*.db")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Копируем содержимое
	written, err := io.Copy(tmpFile, reader)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("copy upload: %w", err)
	}
	tmpFile.Close()

	if written == 0 {
		return fmt.Errorf("empty backup file")
	}

	// Проверяем целостность
	tmpDB, err := sql.Open("sqlite", tmpFile.Name())
	if err != nil {
		return fmt.Errorf("open temp db: %w", err)
	}

	var integrityResult string
	err = tmpDB.QueryRow("PRAGMA integrity_check").Scan(&integrityResult)
	tmpDB.Close()

	if err != nil || integrityResult != "ok" {
		return fmt.Errorf("integrity check failed: %s", integrityResult)
	}

	// Закрываем текущую БД и заменяем файл
	// Примечание: в реальности потребуется перезапуск сервера после восстановления
	// Здесь просто копируем файл поверх текущего
	if err := os.Rename(tmpFile.Name(), dbPath); err != nil {
		return fmt.Errorf("replace database: %w", err)
	}

	return nil
}

// CleanupOldBackups удаляет старые бекапы, оставляя только maxCount последних.
func (s *RootService) CleanupOldBackups(maxCount int) error {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backup dir: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "teblorum_backup_") && strings.HasSuffix(entry.Name(), ".db") {
			backups = append(backups, filepath.Join(s.backupDir, entry.Name()))
		}
	}

	if len(backups) <= maxCount {
		return nil
	}

	// Сортируем по времени создания (старые первые)
	sort.Strings(backups)

	toDelete := len(backups) - maxCount
	for i := 0; i < toDelete; i++ {
		if err := os.Remove(backups[i]); err != nil {
			return fmt.Errorf("remove backup %s: %w", backups[i], err)
		}
	}

	return nil
}
