# Настройка teblorum для production

> teblorum **не слушает HTTPS напрямую**. В production рекомендуется размещать
> за reverse proxy (Caddy или Nginx), который терминирует TLS и проксирует
> трафик на `127.0.0.1:8080`.

---

## Архитектура

```
Internet → Reverse proxy (TLS) → localhost:8080 (teblorum HTTP)
```

| Компонент | Порт | Протокол | Где работает |
|-----------|------|----------|-------------|
| Пользователь | 443 | HTTPS | Внешний интерфейс |
| Reverse proxy | 443 → 8080 | HTTP | Тот же сервер |
| teblorum | 8080 | HTTP (только localhost) | `127.0.0.1:8080` |

---

## 1. Базовая настройка

### 1.1 Деплой через deploy.sh

```bash
# На сервере (Ubuntu 24.04):
git clone <repo> /opt/teblorum-src
cd /opt/teblorum-src
sudo bash deploy/deploy.sh
```

После этого teblorum слушает `127.0.0.1:8080`. Проверить:

```bash
curl -s http://127.0.0.1:8080 | head -5
# Должен вернуться HTML-код ленты
```

### 1.2 Настройка конфига

```bash
sudo nano /opt/teblorum/config.json
```

```json
{
  "db_path": "/opt/teblorum/data/teblorum.db",
  "addr": "127.0.0.1:8080",
  "backup_dir": "/opt/teblorum/backups",
  "session_ttl_days": 30,
  "rate_limit_per_minute": 60,
  "google_client_id": "…",
  "google_client_secret": "…",
  "google_redirect_url": "https://ваш-домен.com/auth/google/callback",
  "bootstrap_email": "admin@example.com",
  "bootstrap_password": "надёжный-пароль"
}
```

```bash
sudo systemctl restart teblorum
```

---

## 2. Reverse proxy: Caddy (рекомендуется)

Caddy — самый простой способ. Автоматический HTTPS через Let's Encrypt.

### Установка

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy
```

### Конфигурация

```bash
sudo nano /etc/caddy/Caddyfile
```

```
ваш-домен.com {
    reverse_proxy 127.0.0.1:8080

    # Статика отдаётся напрямую (опционально, для производительности)
    @static {
        path /static/*
    }
    handle @static {
        root * /opt/teblorum-src/webassets/web/static
        file_server
    }

    # Логи
    log {
        output file /var/log/caddy/teblorum.log
    }
}
```

```bash
sudo caddy fmt /etc/caddy/Caddyfile
sudo systemctl restart caddy
sudo journalctl -u caddy -f
```

### Файрвол (UFW)

```bash
sudo ufw allow 80/tcp    # HTTP (для Let's Encrypt challenge)
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 22/tcp    # SSH
sudo ufw enable
```

### Проверка

```bash
curl -sI https://ваш-домен.com | head -5
# Должен вернуть 200 OK
```

---

## 3. Reverse proxy: Nginx + certbot

Альтернатива Caddy.

### Установка

```bash
sudo apt install nginx certbot python3-certbot-nginx
```

### Конфигурация

```bash
sudo nano /etc/nginx/sites-available/teblorum
```

```nginx
server {
    listen 80;
    server_name ваш-домен.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name ваш-домен.com;

    # SSL — заполнит certbot
    ssl_certificate     /etc/letsencrypt/live/ваш-домен.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/ваш-домен.com/privkey.pem;

    # Безопасность
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Прокси на teblorum
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket (если понадобится)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # Статика напрямую (опционально)
    location /static/ {
        alias /opt/teblorum/static/;
        expires 7d;
        add_header Cache-Control "public, immutable";
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/teblorum /etc/nginx/sites-enabled/
sudo certbot --nginx -d ваш-домен.com
sudo nginx -t && sudo systemctl reload nginx
```

### Файрвол

```bash
sudo ufw allow 'Nginx Full'
sudo ufw allow 22/tcp
sudo ufw enable
```

---

## 4. Мониторинг

### Логи teblorum

```bash
sudo journalctl -u teblorum -f
```

### Логи Caddy

```bash
sudo journalctl -u caddy -f
# или
tail -f /var/log/caddy/teblorum.log
```

### Логи Nginx

```bash
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

### Health check

teblorum не имеет отдельного `/health` эндпоинта. Проверка через HTTP-статус:

```bash
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/
# Должен вернуть 200
```

Можно добавить в systemd-сервис `Restart=on-failure` (уже настроено).

---

## 5. Бекапы

### Автоматические (через root-панель)

```bash
# POST /root/backup создаёт бекап в /opt/teblorum/backups/
```

### Ручной бекап SQLite

```bash
sudo -u teblorum sqlite3 /opt/teblorum/data/teblorum.db ".backup '/opt/teblorum/backups/manual_$(date +%Y%m%d_%H%M%S).db'"
```

### Очистка старых бекапов

Настраивается через `config.json` — `backup_dir`. Старые бекапы удаляются
вручную или через cron, например:

```bash
# Хранить 10 последних бекапов
ls -t /opt/teblorum/backups/*.db | tail -n +11 | xargs rm -f
```

Добавить в cron:

```bash
sudo crontab -e
# Строка:
0 4 * * * ls -t /opt/teblorum/backups/*.db | tail -n +11 | xargs rm -f
```

---

## 6. Обновление

```bash
cd /opt/teblorum-src
git pull
go build -ldflags="-s -w" -o teblorum ./cmd/teblorum/
sudo systemctl stop teblorum
sudo cp teblorum /opt/teblorum/teblorum
sudo systemctl start teblorum
sudo journalctl -u teblorum -f
```

Или через deploy.sh (перезапишет бинарник, конфиг не тронет):

```bash
sudo bash deploy/deploy.sh
```

---

## 7. Переменные окружения (приоритет над config.json)

| Переменная | Описание |
|-----------|----------|
| `TEBLORUM_CONFIG` | Путь к config.json |
| `TEBLORUM_DB_PATH` | Путь к SQLite-файлу |
| `TEBLORUM_ADDR` | Адрес для прослушивания (например, `127.0.0.1:8080`) |
| `TEBLORUM_GOOGLE_CLIENT_ID` | Google OAuth Client ID |
| `TEBLORUM_GOOGLE_CLIENT_SECRET` | Google OAuth Client Secret |
| `TEBLORUM_GOOGLE_REDIRECT_URL` | Redirect URL для Google OAuth |
| `TEBLORUM_BOOTSTRAP_EMAIL` | Email root-пользователя (создаётся при первом запуске) |
| `TEBLORUM_BOOTSTRAP_PASSWORD` | Пароль root-пользователя |
| `TEBLORUM_BACKUP_DIR` | Директория для бекапов |
| `TEBLORUM_SESSION_TTL` | TTL сессии в днях (по умолчанию 30) |
| `TEBLORUM_RATE_LIMIT` | Лимит запросов в минуту (по умолчанию 60) |

---

## 8. Решение проблем

### teblorum не стартует

```bash
sudo journalctl -u teblorum -e
```

### 502 Bad Gateway (Caddy/Nginx)

Проверить, что teblorum запущен:

```bash
curl -s http://127.0.0.1:8080/
```

Если нет:

```bash
sudo systemctl restart teblorum
```

### Permission denied для БД

```bash
sudo chown -R teblorum:teblorum /opt/teblorum/data
```

### Port already in use

```bash
sudo ss -tlnp | grep 8080
# Найти процесс и остановить
```