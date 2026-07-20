---
title: Плагины
description: Расширение Редполитики через gRPC-плагины
weight: 60
lang: ru
---

# Плагины

Редполитика расширяется через gRPC-плагины на базе HashiCorp go-plugin. Плагины запускаются как отдельные процессы и общаются с ядром по gRPC.

## Extension points

Ядро определяет extension points — интерфейсы, которые может реализовать плагин. Плагины не наследуют классы ядра, а реализуют контракты.

Доступные extension points:

| Extension point | Описание |
|---------------|----------|
| `Detector` | Кастомная детекция (не regex/wordlist) |
| `Fixer` | Кастомные фиксы |
| `LLMProvider` | Провайдер LLM (OpenAI, Anthropic и т.д.) |
| `IdentityProvider` | Аутентификация/авторизация |
| `Migrator` | Миграции для EE-функций |
| `Pages` | Система страниц/документации |

## Архитектура

```
┌─────────────┐     gRPC      ┌──────────────┐
│  Go ядро    │◄─────────────►│  Плагин       │
│  (Echo+FX)  │   go-plugin   │  (любой язык) │
└─────────────┘               └──────────────┘
                                   │
                                   ▼
                          ┌──────────────────┐
                          │  Своя логика     │
                          │  (LLM, NER, POS) │
                          └──────────────────┘
```

Плагин — отдельный бинарник, который ядро запускает как дочерний процесс. go-plugin управляет жизненным циклом: запуск, health-check, graceful shutdown.

## Языки

- **Go** — пишите плагины как Go-модули, используйте ту же proto
- **Не Go** — используйте plugin-sdk (отдельное репо), реализуйте gRPC-сервер на своём языке
- **Архетипов нет** — ядро не знает «NERPlugin» или «POSPlugin», только extension points

## Пример: плагин pages

Встроенный CE-плагин, который читает markdown-файлы с frontmatter и отдаёт через API:

```bash
# Бинарник встроен в образ
/etc/redpolitika/plugins/redpolitika-pages --pages-dir=/etc/redpolitika/pages
```

Конфигурация:

```yaml
# docker-compose.yml
environment:
  PLUGIN_PAGES_FLAGS: "--pages-dir=/etc/redpolitika/pages"
volumes:
  - ./pages:/etc/redpolitika/pages:ro
```

## Конфигурация плагинов

Плагины ищутся в `PLUGINS_DIR` (`/etc/redpolitika/plugins`). Каждый бинарник запускается как отдельный плагин. Флаги передаются через переменную `PLUGIN_<NAME>_FLAGS`.

CE-плагины собираются в образ на этапе Docker-сборки. Для собственных плагинов монтируйте директорию с бинарниками.

## Hot-reload

Hot-reload не поддерживается — для обновления плагина нужен перезапуск контейнера.

## Proto-контракты

Определения proto находятся в `backend/proto/` и `ce-plugins/pages/proto/`. Ключевые:

- `proto/detect/detect.proto` — интерфейс детектора
- `proto/fix/fix.proto` — интерфейс фикса
- `proto/llm/llm.proto` — LLM-провайдер
- `proto/pages/pages.proto` — система страниц
- `proto/identity/identity.proto` — аутентификация
- `proto/migrator/migrator.proto` — миграции

## Связанное

- [Развёртывание](guide-deployment.md) — конфигурация плагинов в Docker
- [API](guide-api.md) — endpoints
- [Правила](guide-rules.md) — YAML-формат
