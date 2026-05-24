package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config содержит все настройки приложения.
type Config struct {
	// База данных
	DBPath string `json:"db_path"`

	// HTTP-сервер
	Addr string `json:"addr"`

	// Google OAuth
	GoogleClientID     string `json:"google_client_id"`
	GoogleClientSecret string `json:"google_client_secret"`
	GoogleRedirectURL  string `json:"google_redirect_url"`

	// Root-пользователь (bootstrap)
	BootstrapEmail    string `json:"bootstrap_email"`
	BootstrapPassword string `json:"bootstrap_password"`

	// Бекапы
	BackupDir string `json:"backup_dir"`

	// Сессии
	SessionTTLDays int `json:"session_ttl_days"`

	// Rate limiting
	RateLimitPerMinute int `json:"rate_limit_per_minute"`

	// TLS
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`
	TLSDomain   string `json:"tls_domain"` // для autocert
	UseAutocert bool   `json:"use_autocert"`
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() *Config {
	return &Config{
		DBPath:             "teblorum.db",
		Addr:               ":8080",
		BackupDir:          "./backups",
		SessionTTLDays:     30,
		RateLimitPerMinute: 60,
	}
}

// LoadConfig загружает конфигурацию из файла и переменных окружения.
// Переменные окружения имеют приоритет над файлом.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Загрузка из файла (если существует)
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config file: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	// Переменные окружения (приоритет)
	envOverrides := map[string]*string{
		"TEBLORUM_DB_PATH":              &cfg.DBPath,
		"TEBLORUM_ADDR":                 &cfg.Addr,
		"TEBLORUM_GOOGLE_CLIENT_ID":     &cfg.GoogleClientID,
		"TEBLORUM_GOOGLE_CLIENT_SECRET": &cfg.GoogleClientSecret,
		"TEBLORUM_GOOGLE_REDIRECT_URL":  &cfg.GoogleRedirectURL,
		"TEBLORUM_BOOTSTRAP_EMAIL":      &cfg.BootstrapEmail,
		"TEBLORUM_BOOTSTRAP_PASSWORD":   &cfg.BootstrapPassword,
		"TEBLORUM_BACKUP_DIR":           &cfg.BackupDir,
		"TEBLORUM_TLS_CERT":             &cfg.TLSCertFile,
		"TEBLORUM_TLS_KEY":              &cfg.TLSKeyFile,
		"TEBLORUM_TLS_DOMAIN":           &cfg.TLSDomain,
	}

	for env, field := range envOverrides {
		if val := os.Getenv(env); val != "" {
			*field = val
		}
	}

	// Числовые переменные окружения
	if val := os.Getenv("TEBLORUM_SESSION_TTL"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cfg.SessionTTLDays = v
		}
	}
	if val := os.Getenv("TEBLORUM_RATE_LIMIT"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cfg.RateLimitPerMinute = v
		}
	}
	if os.Getenv("TEBLORUM_USE_AUTOCERT") == "true" {
		cfg.UseAutocert = true
	}

	return cfg, nil
}

// SaveConfig сохраняет конфигурацию в файл.
func SaveConfig(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
