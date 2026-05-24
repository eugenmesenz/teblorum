Запуск проекта локально (HTTP, без TLS)

1) Установите Go (версия, указанная в go.mod).

2) Настройка (необязательно)
- Отредактируйте `config/local.json` при необходимости (порт, путь к БД и т.д.).
- При желании создайте файл `.env` на основе `.env.example` и загрузите его в среду.

3) Быстрый запуск в foreground
```bash
./scripts/start_local.sh
```

4) Запуск в background (nohup)
```bash
./scripts/start_local_bg.sh
# лог: teblorum.log, pid: teblorum.pid
```

5) Проверка
- Откройте http://localhost:8080 (или другой адрес из `config/local.json`).

6) Примечания
- Скрипты собирают бинарь в `bin/teblorum` и создают `data/` и `backups/`.
- Для production используйте TLS и reverse proxy (см. `deploy/PRODUCTION.md`).
