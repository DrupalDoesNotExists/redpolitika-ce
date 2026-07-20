---
title: API
description: REST + WebSocket endpoints, примеры запросов, формат ошибок
weight: 40
language: ru
---

# API

Базовый URL: `http://localhost:8080`

---

## REST endpoints

### `GET /health` / `GET /healthz`

```json
{
  "status": "ok",
  "plugins": ["redpolitika-ner"]
}
```

`plugins` — список запущенных плагинов (пустой, если нет).

### `GET /metrics`

Prometheus text exposition (`redpolitika_analyze_total`, `redpolitika_analyze_latency_seconds`, `redpolitika_flags_total`).

### `GET /version`

Метаданные сборки (через `-ldflags` / Docker build-args, не хардкод).

```json
{
  "version": "v0.1.0b",
  "commit": "a43c3b4",
  "build_time": "2026-07-19T10:40:00Z",
  "license": "BSL-1.1",
  "module": "ce",
  "component": "redpolitika"
}
```

Локальная `go build` без ldflags: `version=dev`, `commit=unknown`, `build_time=unknown`.

### `GET /api/pages`

Список всех страниц документации. Используется фронтендом для построения каталога.

### `GET /api/pages/:slug`

Контент страницы документации. `slug` — путь без расширения `.md`. Пример: `/api/pages/docs/overview`.

### `GET /api/rules`

Все загруженные правила с метаданными.

```json
[
  {
    "id": "cleanliness/obscene",
    "severity": 8,
    "category": "cleanliness",
    "detect_method": "wordlist",
    "suggestion": "Заменить на нейтральное выражение",
    "auto_fix": "***",
    "client_side": true,
    "name": "Обсценная лексика",
    "url": "/rules/obscene",
    "examples": {
      "bad": ["Ты дурак"],
      "good": ["Вы неправы"]
    },
    "related": [
      {"name": "Грубые выражения", "url": "/rules/rude"}
    ]
  }
]
```

### `GET /api/client-rules`

Только client-side правила — regex/wordlist листья, которые фронтенд может применить локально.

```json
[
  {
    "id": "cleanliness/obscene",
    "severity": 8,
    "category": "cleanliness",
    "method": "wordlist",
    "pattern": "",
    "words": ["дурак", "идиот", "кретин"],
    "case_sensitive": false,
    "suggestion": "Заменить на нейтральное выражение",
    "auto_fix": "***",
    "engine": "",
    "name": "Обсценная лексика",
    "url": "/rules/obscene",
    "examples": {
      "bad": ["Ты дурак"],
      "good": ["Вы неправы"]
    },
    "related": [
      {"name": "Грубые выражения", "url": "/rules/rude"}
    ]
  }
]
```

Поля зависят от метода детекции:
- `method: "regex"` — включает `pattern` (string), `engine: "re2"`
- `method: "wordlist"` — включает `words` (array), `case_sensitive` (bool)

### `POST /api/analyze`

Анализ текста.

Параметры запроса:
- `?full=true` — включить client-side правила (по умолч. только server-side)

Запрос:

```json
{
  "text": "Ты дурак. Это очень важно."
}
```

Ответ:

```json
{
  "flags": [
    {
      "id": "a1b2c3d4e5f6g7h8",
      "rule_id": "cleanliness/obscene",
      "category": "cleanliness",
      "severity": 8,
      "message": "Match found: 'дурак'",
      "suggestion": "Заменить на нейтральное выражение",
      "auto_fix": "***",
      "anchor": {
        "paragraph_index": 0,
        "occurrence": 0,
        "match_text": "дурак"
      },
      "state": "pending",
      "rule_name": "Обсценная лексика",
      "rule_url": "/rules/obscene",
      "examples": {
        "bad": ["Ты дурак"],
        "good": ["Вы неправы"]
      },
      "related": [
        {"name": "Грубые выражения", "url": "/rules/rude"}
      ]
    }
  ],
  "cleanliness_score": 8.5,
  "readability_score": 10.0,
  "flag_count": 1,
  "session_id": ""
}
```

---

## WebSocket — `/ws/live`

Двунаправленный JSON-фреймовый протокол.

### Клиент → Сервер

#### check / analyze

```json
{"type": "check", "text": "Текст для анализа", "textHash": "", "full": false}
```

- `type: "check"` и `type: "analyze"` идентичны по поведению
- Сервер debounce 500ms

#### accept / reject

```json
{"type": "accept", "flagId": "a1b2c3d4e5f6g7h8"}
{"type": "reject", "flagId": "a1b2c3d4e5f6g7h8"}
```

#### applyAll

```json
{"type": "applyAll", "flagIds": ["a1b2c3d4e5f6g7h8", "i9j0k1l2m3n4o5p6"]}
```

### Сервер → Клиент

#### check_result

```json
{
  "type": "check_result",
  "textHash": "",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "flags": [...],
  "scores": {
    "cleanliness": 8.5,
    "readability": 10.0
  }
}
```

#### ack

```json
{"type": "ack", "action": "accept", "flagId": "a1b2c3d4e5f6g7h8", "status": "ok"}
{"type": "ack", "action": "applyAll", "flagIds": ["a1b2c3d4e5f6g7h8"], "status": "ok"}
{"type": "ack", "action": "reject", "status": "error", "error": "invalid flagId"}
```

---

## Формат ошибок

Ошибки в формате RFC 7807 (Problem Details):

```json
{
  "type": "/errors/invalid-rule",
  "title": "Invalid rule",
  "status": 422,
  "detail": "regex pattern contains invalid syntax"
}
```

---

## Оценки

- **Cleanliness** — 0–10, штраф от флагов с `category: "cleanliness"`
- **Readability** — 0–10, штраф от флагов с `category: "readability"`
- Нормализация на 100 слов
- Каждый флаг вычитает своё значение severity
- Только pending (непринятые) флаги влияют на оценку

## Формат флага

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | string | FNV-1a 64 от `rule_id + match_text + paragraph_index + occurrence_in_paragraph` |
| `rule_id` | string | ID сработавшего правила |
| `category` | string | Категория |
| `severity` | int | 1–10 |
| `message` | string | Описание срабатывания |
| `suggestion` | string | Рекомендация |
| `auto_fix` | string/null | Текст автозамены |
| `anchor` | object | `paragraph_index`, `occurrence`, `match_text` |
| `state` | string | `pending` / `accepted` / `rejected` |
| `rule_name` | string | Человеческое название правила |
| `rule_url` | string | Ссылка на документацию правила |
| `examples` | object | Примеры bad/good |
| `related` | array | Связанные правила |

## Связанное

- [Правила](guide-rules.md) — как создавать правила
- [Рецепты](cookbook.md) — готовые паттерны
- [Развёртывание](guide-deployment.md) — конфигурация сервера
