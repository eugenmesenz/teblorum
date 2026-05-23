package repo

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"strings"
)

//go:embed sql/*.sql
var migrationFS embed.FS

// migrations — список миграций в порядке применения.
// Каждый элемент: номер версии → имя файла с SQL.
var migrations = []struct {
	version  int
	filename string
}{
	{1, "sql/001_init.sql"},
	{2, "sql/002_password_resets.sql"},
}

// latestSchemaVersion — максимальная версия схемы БД.
const latestSchemaVersion = 2

// RunMigrations применяет все неприменённые миграции к базе данных.
func RunMigrations(db *sql.DB) error {
	// Создаём таблицу schema_migrations, если её нет
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Проверяем текущую версию
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("read current migration version: %w", err)
	}

	if currentVersion >= latestSchemaVersion {
		return nil
	}

	// Применяем миграции последовательно
	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}
		log.Printf("applying migration %d (%s)", m.version, m.filename)

		sqlBytes, err := migrationFS.ReadFile(m.filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.filename, err)
		}

		// Выполняем каждое выражение отдельно
		statements := strings.Split(string(sqlBytes), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("execute migration %d statement: %w (stmt: %s)",
					m.version, err, stmt[:min(len(stmt), 80)])
			}
		}

		// Записываем версию
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version)
		if err != nil {
			return fmt.Errorf("record migration version %d: %w", m.version, err)
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
