#!/bin/bash
# Скрипт деплоя teblorum
# Запускать от root или через sudo

set -euo pipefail

APP_NAME="teblorum"
APP_USER="teblorum"
APP_DIR="/opt/${APP_NAME}"
BINARY="${APP_DIR}/${APP_NAME}"
SERVICE_FILE="deploy/${APP_NAME}.service"
CONFIG_FILE="${APP_DIR}/config.json"
DATA_DIR="${APP_DIR}/data"
BACKUP_DIR="${APP_DIR}/backups"

echo "=== Деплой ${APP_NAME} ==="

# 1. Сборка бинарника
echo "[1/5] Сборка бинарника..."
go build -ldflags="-s -w" -o "${APP_NAME}" ./cmd/${APP_NAME}/
echo "  OK: ${APP_NAME} собран"

# 2. Создание пользователя и директорий
echo "[2/5] Настройка директорий..."
if ! id -u ${APP_USER} >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin ${APP_USER}
    echo "  Создан пользователь ${APP_USER}"
fi

mkdir -p ${APP_DIR} ${DATA_DIR} ${BACKUP_DIR}
echo "  Директории созданы"

# 3. Копирование файлов
echo "[3/5] Копирование файлов..."
cp ${APP_NAME} ${BINARY}
chmod 755 ${BINARY}

if [ -f "${CONFIG_FILE}" ]; then
    echo "  Конфиг уже существует, пропускаем"
else
    cat > ${CONFIG_FILE} << 'EOF'
{
  "db_path": "/opt/teblorum/data/teblorum.db",
  "addr": "127.0.0.1:8080",
  "backup_dir": "/opt/teblorum/backups",
  "session_ttl_days": 30,
  "rate_limit_per_minute": 60
}
EOF
    echo "  Создан конфиг по умолчанию"
fi

chown -R ${APP_USER}:${APP_USER} ${APP_DIR}
echo "  Права установлены"

# 4. Установка systemd unit
echo "[4/5] Установка systemd unit..."
cp ${SERVICE_FILE} /etc/systemd/system/${APP_NAME}.service
systemctl daemon-reload
echo "  Unit установлен"

# 5. Запуск
echo "[5/5] Запуск сервиса..."
systemctl enable ${APP_NAME}
systemctl restart ${APP_NAME}
systemctl status ${APP_NAME} --no-pager

echo ""
echo "=== Деплой завершён ==="
echo "Проверка: journalctl -u ${APP_NAME} -f"