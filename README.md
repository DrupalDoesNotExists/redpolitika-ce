---
title: Редполитика β
description: Open-core сервис проверки и правки текста по правилам редполитики в YAML
lang: ru
---

# Redpolitika

**Open-core сервис проверки и правки текста по правилам редакционной политики в YAML.**  
Аналог «Главреда», но под любую редполитику — для любого языка.

Определите стиль редполитики YAML-правилами — Redpolitika проверяет текст, находит нарушения и предлагает исправления. Работает сразу с Docker Compose.

---

## Возможности

- **YAML-правила** — композируемые деревья detect/fix, 20+ методов детекции и 15+ методов фиксов
- **Два скоринга** — чистота и читаемость (0–10), нормализация на 100 слов
- **WebSocket live** — флаги в реальном времени по мере набора текста
- **CodeMirror 6** — inline-отображение флагов в редакторе
- **Extension points** — gRPC-плагины для LLM, NER, POS-теггеров (WIP: скоро будут плагины для NLP и ED)
- **Self-hosted** — полный контроль над данными, без отправки текстов вовне
- **Слои правил** — base → project → override с deep-merge по id

## Быстрый старт

```bash
mkdir -p deploy/rules deploy/plugins
docker compose -f deploy/docker-compose.yml up --build
```

Откройте `http://localhost:8080`. Поместите YAML-файлы правил в `deploy/rules/`.

### GHCR (готовый образ)

```bash
mkdir -p rules
docker run --rm -p 8080:8080 \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  ghcr.io/drupaldoesnotexists/redpolitika-ce:latest
```

## Архитектура

```
backend/        Go (Echo, Uber FX, DDD / ports-and-adapters)
frontend/       Next.js (standalone SSR, Caddy reverse proxy)
ce-plugins/     Встроенные плагины CE (Python, Go)
plugin-sdk/     SDK для плагинов не на Go
deploy/         Dockerfile + docker-compose.yml
```

Стек: Go, Echo, Uber FX, SQLite/Postgres, Next.js, CodeMirror 6, gRPC, Tailwind + shadcn/ui.

## Правила

Правила задаются в YAML с композируемым деревом detect/fix:

```yaml
rules:
  - id: typography/ellipsis
    severity: 5
    category: readability
    detect:
      regex: "\\.\\.\\."
    fix:
      replace:
        with: "…"
    suggestion: Замените три точки на символ многоточия
```

Три слоя правил: `base → project → override` с deep-merge по `id`.

## Документация

- [Обзор](/pages/docs/overview) — архитектура и концепция
- [Быстрый старт](/pages/docs/quickstart) — запуск за 5 минут
- [Правила](/pages/docs/guide-rules) — полный формат YAML-правил
- [API](/pages/docs/guide-api) — REST + WebSocket
- [Развёртывание](/pages/docs/guide-deployment) — Docker, конфигурация
- [Плагины](/pages/docs/guide-plugins) — расширения через gRPC
- [Рецепты](/pages/docs/cookbook) — готовые паттерны правил

## Лицензия

**Business Source License 1.1** — см. [LICENSE](LICENSE).

- **Change Date:** 2030-07-18 → Apache 2.0
- **Additional Use Grant:** бесплатно для некоммерческого использования и небольших организаций:
  - ≤ 15 млн ₽ годовой выручки (юрлица РФ)
  - ≤ $400K годовой выручки (остальной мир)
  - Два независимых фиксированных порога — без привязки курсом

Enterprise-функции (EE) проприетарны и поставляются отдельно.

## Уведомления

См. [NOTICE](NOTICE) о сторонних лицензиях.
