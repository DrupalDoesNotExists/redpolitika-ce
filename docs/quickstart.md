---
title: Быстрый старт
description: Запуск Редполитики за 5 минут
weight: 20
lang: ru
---

# Быстрый старт

## 1. Запуск через Docker Compose

```bash
mkdir -p deploy/rules deploy/plugins
docker compose -f deploy/docker-compose.yml up --build
```

Откройте `http://localhost:8080`. Сервер работает, но без правил не показывает флаги.

## 2. Добавьте правила

Создайте `deploy/rules/typography.yaml`:

```yaml
rules:
  - id: "typography/ellipsis"
    severity: 5
    category: "readability"
    name: "Многоточие"
    detect:
      regex: "\\.\\.\\."
    fix:
      replace:
        with: "…"
    suggestion: "Замените три точки на символ многоточия"
```

Перезапустите: `docker compose -f deploy/docker-compose.yml restart redpolitika`.

## 3. Проверьте работу

```bash
curl -X POST http://localhost:8080/api/analyze \
  -H 'Content-Type: application/json' \
  -d '{"text":"Это очень важно..."}'
```

В ответе — флаг с `rule_id: "typography/ellipsis"`.

## 4. Запуск без Docker

### Backend

```bash
cd backend
go build -o redpolitika ./cmd/api/
RULES_DIR=../deploy/rules STATIC_DIR=../frontend/out ./redpolitika
```

### Frontend (разработка)

```bash
cd frontend
npm install
npm run dev
```

Фронтенд на `http://localhost:3000`, проксирует API на `http://localhost:8080`.

## 5. GHCR (готовый образ)

```bash
mkdir -p rules
docker run --rm -p 8080:8080 \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  ghcr.io/drupaldoesnotexists/redpolitika-ce:latest
```

Образ включает UI и API. Правила не встроены — монтируйте свою директорию.

## Дальнейшие шаги

- [Полный гайд по правилам](guide-rules.md) — все методы детекции и фиксов
- [Развёртывание](guide-deployment.md) — production-конфигурация
- [Рецепты](cookbook.md) — готовые паттерны правил
