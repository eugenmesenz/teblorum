package repo

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
)

//go:embed sql/001_init.sql
var migrationFS embed.FS

const schemaVersion = 1

// RunMigrations применяет миграции к базе данных.
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

	if currentVersion >= schemaVersion {
		return nil // уже применено
	}

	// Читаем SQL-миграцию
	sqlBytes, err := migrationFS.ReadFile("sql/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	// Выполняем каждое выражение отдельно
	statements := strings.Split(string(sqlBytes), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("execute migration statement: %w (stmt: %s)", err, stmt[:min(len(stmt), 80)])
		}
	}

	// Записываем версию
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", schemaVersion)
	if err != nil {
		return fmt.Errorf("record migration version: %w", err)
	}

	return nil
}

// min — вспомогательная функция
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
