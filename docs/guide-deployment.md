---
title: Развёртывание
description: Docker, конфигурация, переменные окружения, сборка из исходников
weight: 50
lang: ru
---

# Развёртывание

## Запуск из GHCR

```bash
docker run --rm -p 8080:8080 \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  ghcr.io/drupaldoesnotexists/redpolitika-ce:latest
```

Откройте `http://localhost:8080`.

Образ включает UI и API. Правила **не встроены** — монтируйте директорию с YAML в `/etc/redpolitika/rules` (можно пустую: сервер запустится, флагов не будет до добавления правил).

### Персистентность SQLite

```bash
docker run --rm -p 8080:8080 \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  -v redpolitika-data:/data \
  ghcr.io/drupaldoesnotexists/redpolitika-ce:latest
```

## Docker Compose (локальная сборка)

```bash
mkdir -p deploy/rules deploy/plugins
docker compose -f deploy/docker-compose.yml up --build
```

Откройте `http://localhost:8080`.

## Переменные окружения

У всех есть значения по умолчанию — переопределяйте только нужное.

| Переменная | Значение по умолч. | Описание |
|-----------|-------------------|----------|
| `PORT` | `8081` | Порт HTTP (Go-бэкенд; Caddy — входной reverse proxy на :8080) |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `ENVIRONMENT` | `production` | Метка окружения |
| `STATIC_DIR` | `../frontend/out` | Директория статики Next.js (dev; в production Caddy → Next.js) |
| `RULES_DIR` | `/etc/redpolitika/rules` | Базовый слой YAML-правил |
| `RULES_PROJECT_DIR` | — | Проектный слой (override) |
| `RULES_OVERRIDE_DIR` | — | Финальный слой (per-env override) |
| `PLUGINS_DIR` | `/etc/redpolitika/plugins` | Директория бинарников плагинов |
| `DB_DRIVER` | `sqlite` | `sqlite` или `postgres` |
| `DB_DSN` | `file:/data/redpolitika.db?…` | Строка подключения к БД |
| `PARAGRAPH_SEPARATOR` | — | Разделитель абзацев (пустая строка → `\n\n`) |
| `PLUGIN_PAGES_FLAGS` | `--pages-dir=…` | Флаги для плагина pages |

### PostgreSQL

```bash
docker run --rm -p 8080:8080 \
  -e DB_DRIVER=postgres \
  -e DB_DSN="postgres://user:password@db:5432/redpolitika?sslmode=disable" \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  ghcr.io/drupaldoesnotexists/redpolitika-ce:latest
```

Миграции применяются автоматически при первом запуске.

## Сборка из исходников

### Backend (Go)

```bash
cd backend
go build -o redpolitika ./cmd/api/
RULES_DIR=../deploy/rules STATIC_DIR=../frontend/out ./redpolitika
```

### Frontend

```bash
cd frontend
npm install
npm run build    # → frontend/out/
```

### Сборка образа

```bash
docker build -f deploy/Dockerfile -t redpolitika-ce:local .
docker run --rm -p 8080:8080 \
  -v "$PWD/my-rules:/etc/redpolitika/rules:ro" \
  redpolitika-ce:local
```

## CI-релизы (GHCR)

При пуше git-тега `v*` workflow `.github/workflows/release.yml` собирает и пушит:

- `ghcr.io/<owner>/redpolitika-ce:latest`
- `ghcr.io/<owner>/redpolitika-ce:<version>`


Аутентификация через `GITHUB_TOKEN` (`packages: write`). После первого пуша установите видимость пакета в **public** в настройках GitHub Packages, если нужен анонимный `docker pull`.

## Структура директорий

```
deploy/
├── pages/               # Документация (монтируется в плагин pages)
│   └── docs/
├── plugins/             # Бинарники плагинов (из образа)
├── rules/               # Ваши YAML-правила
├── Caddyfile
├── Dockerfile
├── docker-compose.yml
└── entrypoint.sh
```

## Связанное

- [Правила](guide-rules.md) — YAML-формат
- [Рецепты](cookbook.md) — готовые паттерны
- [API](guide-api.md) — REST + WebSocket
- [Плагины](guide-plugins.md) — расширения через gRPC
