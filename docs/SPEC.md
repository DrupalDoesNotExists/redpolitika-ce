# SPEC.md — Редполитика

> Техническая спецификация CE-ядра Редполитики. Описывает архитектуру, доменную модель, протоколы, плагин-систему и дизайн-решения.

---

## 1. Назначение и концепция

**Редполитика** — open-core сервис проверки и правки текста по гибким правилам редакционной политики (редполитики) в YAML. Концептуально — «Главред, но для любой редполитики»: правила описываются в YAML, а не зашиты в код.

### Границы CE-версии

- Self-host (первичная цель); архитектурно не привязан к self-host
- Один проект / один пользователь, без авторизации и RBAC
- Правила и документы не хранятся в БД (YAML на диске, текст — в live-сессии)
- LLM — только через стороннего провайдера, опционально через плагин
- NER и POS — вынесены в отдельные CE-плагины

### Вне CE (EE-репо)

RBAC, SSO/SAML/OIDC, on-prem нейросети, мультитенантность, хранение документов, enterprise-плагины, aaS-режим. CE-ядро проектируется так, чтобы EE-слой надстраивался через extension points без переписывания.

---

## 2. Глоссарий

| Термин | Определение |
|--------|------------|
| Правило (rule) | YAML-описание: стадия обнаружения + метаинформация + опциональное автоисправление |
| Флаг (flag) | Факт срабатывания правила на фрагменте текста |
| Клиентское правило | Правило, чья цепочка обнаружения использует только regex/wordlist простые методы; исполняется на фронтенде |
| Серверное правило | Требует LLM/NER/POS/плагин; исполняется на бэкенде |
| Плагин | Внешний бинарник, запускаемый ядром по go-plugin-протоколу |
| CE-плагин | Плагин, входящий в CE-дистрибутив |
| Extension point | gRPC-интерфейс, который может реализовать плагин |
| Session | WebSocket-сессия анализа текста, in-memory |
| Score | Числовая оценка текста (0–10), две категории: чистота и читаемость |

---

## 3. Высокоуровневая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│  Браузер (Next.js клиент)                                    │
│  - CodeMirror 6 (plain) с live-подсветкой (decorations)      │
│  - исполняет клиентские правила локально                     │
│  - WebSocket: текст → бэкенд, флаги ← бэкенд                 │
│  - принять/отклонить предложение, «применить всё»            │
└───────────────┬─────────────────────────────────────────────┘
        REST (control) │ WebSocket (live)
┌───────────────▼─────────────────────────────────────────────┐
│  Backend (Go, Echo, Uber FX)                                 │
│  ┌────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │ Rules Load │  │ Rule Engine  │  │ Plugin Manager     │   │
│  │ (YAML,     │→ │ (обнаружение │→ │ (go-plugin, gRPC)  │   │
│  │  deep-merge)│  │  + фиксы)   │  │  ┌─ NER plugin    │   │
│  └────────────┘  └──────────────┘  │  ├─ POS plugin    │   │
│         │                │         │  └─ Pages plugin  │   │
│         │          ┌─────▼──────┐  └────────│───────────┘   │
│         │          │Classifier │           │ gRPC          │
│         │          │(client-   │           ▼               │
│         │          │ capable?)  │     плагин-бинарники       │
│         │          └─────┬──────┘     (Python/Rust/Go)       │
│         │                │                                    │
│         ▼                ▼                                    │
│  /api/client-rules  WebSocket hub                             │
│  (только клиентские)  + debounce 500ms + кэш по хэшу         │
│                                                              │
│  DB layer (задел, ядро НЕ использует):                      │
│  Postgres (pgx v5)  ⇄  SQLite (file/memory)                  │
│  миграции из коробки на старте (для плагинов)               │
└─────────────────────────────────────────────────────────────┘
```

### Поток live-проверки

1. Клиент вводит текст → debounce 500мс → WebSocket отправляет полный текст + хэшсумму
2. Бэкенд проверяет кэш по `(textHash, configHash)`; попадание → закэшированные флаги
3. Промах → Rule Engine прогоняет все правила; клиентские уже могли сработать в браузере, но бэкенд считает всё (canonical)
4. Серверные стадии дёргают плагины по gRPC (LLM, NER, POS)
5. Флаги с хэш-ID стримятся обратно; клиент мёрджит со своими локальными
6. Принять/отклонить — клиент; «применить всё» — применяет все нерассмотренные предложения с автоисправлением

---

## 4. Стек технологий

| Компонент | Технология |
|-----------|-----------|
| Backend | Go, Echo, Uber FX (DI), zap (логирование) |
| Frontend | Next.js (standalone SSR), CodeMirror 6, Tailwind + shadcn/ui |
| State | zustand, react-query |
| Database | SQLite (default) / PostgreSQL (pgx v5), один движок в runtime |
| Plugins | HashiCorp go-plugin, gRPC |
| Protocols | REST (control), WebSocket (live), gRPC (plugins) |
| Infrastructure | Docker, Docker Compose, Caddy (reverse proxy), supervisord |

### Go-зависимости

- Веб-фреймворк: Echo
- DI: Uber FX
- БД: pgx v5 (Postgres), modernc.org/sqlite (SQLite, pure Go, без CGO)
- Плагины: HashiCorp go-plugin
- gRPC: стандартный protobuf + grpc
- Логи: zap
- Миграции: custom dbinfra.Migrator (свой, не библиотечный)
- ULID: oklog/ulid

---

## 5. Структура монорепо

```
redpolitika/
├── backend/                    # Go-ядро
│   ├── cmd/api/                # main.go + wiring DI (Uber FX)
│   ├── internal/
│   │   ├── domain/             # чистый домен
│   │   │   ├── model/          # entity, VO, microtypes
│   │   │   ├── service/        # domain services
│   │   │   └── ports/          # interfaces (repository + extension points)
│   │   ├── usecase/            # application use-cases
│   │   ├── infra/              # adapters: rules, cache, session, db, plugin, llm
│   │   └── transport/          # REST + WebSocket (Echo handlers)
│   └── proto/                  # gRPC-контракты плагинов
├── frontend/                   # Next.js standalone SSR
│   ├── app/
│   ├── components/             # shadcn/ui + custom
│   └── lib/
│       ├── api/                # REST + WS clients
│       ├── client-rule-engine/ # клиентская проверка + CodeMirror decorations
│       └── hooks/
├── ce-plugins/                 # CE-плагины (pages, и т.д.)
├── plugin-sdk/                 # SDK для не-Go плагинов
├── deploy/
│   ├── Dockerfile              # цельный образ
│   └── docker-compose.yml
└── docs/                       # Документация + этот файл
```

### Layer dependency (DDD)

```
domain (model + ports)
    ↑
usecase (знает domain + ports)
    ↑
infra (реализует ports — adapters)
transport (вызывает use-cases)
```

- `domain` не зависит ни от кого
- `usecase` видит только `domain`
- `infra` реализует domain-ports
- `transport` вызывает use-cases
- Слои не протекают друг в друга

---

## 6. Доменная модель (DDD)

### Принципы

Rich domain — поведение живёт методами сущностей, не анемичными data-bags. Домен про инфраструктуру не знает ничего: ни БД, ни диска, ни плагинов, ни транспорта. Репозитории и extension points — это порты (interfaces в домене); их реализации — adapters в инфра-слое. Валидация в конструкторах — невалидные состояния невозможны в памяти.

### Value Objects (immutable, equal-by-value)

- `Text` — plain text + word_count; методы `Tokens()`, `WordCount()`, `Paragraphs()`
- `Score` — (cleanliness, readability), 0–10; factory `Score.from(flags, wordCount)`
- `Span` — (start, end), нормализован; `Overlaps(other)`, `Contains(idx)`
- `MatchText`, `TextHash`, `ConfigHash` — простые VO

### Микротипы (branded strings/ints)

`RuleID`, `FlagID`, `SessionID` (ULID), `Severity` (int 1–10), `Category` (cleanliness|readability), `ParagraphIndex`, `Occurrence`, `WordCount`

### Entities

**`Rule`** — aggregate root. Identity = `RuleID` (из YAML). Invariants: `severity ∈ [1,10]`, detect из допустимого множества, fix совместим с detect, regex RE2-subset. Immutable после загрузки.
- Методы: `Detect(text) []Flag`, `Fix(flag) Suggestion`, `IsClientSide() bool`, `Validate() error`
- Поле `priority` (int, по умолчанию 0) — выше число = раньше применяется

**`Flag`** — entity внутри `Session`. Identity = `FlagID` (FNV-1a 64). Lifecycle: `raised → accepted | rejected | applied`.
- Методы: `Accept()`, `Reject()`, `Apply()`, `IsPending()`

**`RuleSet`** — отдельная сущность (не просто `[]Rule`). Identity = `ConfigHash`. Immutable snapshot после deep-merge.
- Методы: `Merge(other) RuleSet`, `Validate() error`, `Hash() ConfigHash`, `ClientRules()`, `ServerRules()`

### Aggregates

**`Session`** — aggregate root. Identity = `SessionID` (ULID). Содержит `Text`, `ConfigHash`, набор `Flag` (с accept-статусом), `Score`.
- Методы: `AcceptFlag(id)`, `RejectFlag(id)`, `ApplyFix(id)`, `RecomputeScore()`
- Lifecycle = WS-соединение, in-memory, теряется при рестарте

**`Analysis`** — immutable DTO-snapshot одного прогона движка. Ключ кэша = `(TextHash, ConfigHash)`.

### Domain services (pure, без I/O)

- **`RuleEngine`** — гонит `RuleSet` по `Text` → поток `Flag`
- **`LLMBatcher`** — (plan) собирает LLM-правила в один batch-запрос (провайдер = порт `LLMProvider`)
- **`FixApplier`** — применяет fix, резолвит конфликты Span по severity
- **`ScoreCalculator`** — `flags → Score`, нормализация per-100-words

### Use-cases (оркестрируют domain + ports)

- **`LoadRules`** — через `RuleRepository` читает YAML, deep-merge, `RuleSet.Validate()`
- **`AnalyzeText`** — `RuleSet → RuleEngine → ScoreCalculator → Analysis`
- **`AcceptRejectFlag`** — грузит `Session`, вызывает `AcceptFlag/RejectFlag`
- **`ApplyFix`** — `Session.ApplyFix` / «применить всё»
- **`RegisterPlugin`** — инфра: загрузка, handshake, регистрация adapters

### Ports ↔ Adapters

- **Репозитории:** `RuleRepository` (YAML → deep-merge), `SessionRepository` (in-memory WS-bound), `CacheRepository` (bounded LRU+TTL `(TextHash, ConfigHash) → Analysis`)
- **Extension points:** `DetectFunctionProvider`, `FixFunctionProvider`, `LLMProvider`, `Migrator`, `FrontendBundleProvider`, `BrandingProvider`, `StorageProvider`, `AuthProvider`, `RolePermissionProvider`, `BillingProvider`, `DocumentStorageBackend`, `AuditLogger`, `MetricsExporter`, `WebhookProvider`, `RuleValidator`
- Все extension points — interfaces в домене; реализации — gRPC-плагины (adapters)

### Доменные события

В CE нет персистентности и потребителей. EE добавит `AuditLogger` / `WebhookProvider` как потребителей событий.

---

## 7. Правила — YAML-схема и движок

### Формат файла

```yaml
rules:
  - id: "category/name"
    severity: 5                # 1–10
    category: "cleanliness"    # cleanliness | readability | custom
    name: "Человеческое название"
    url: "/rules/name"
    suggestion: "Что делать"
    detect:
      # дерево методов
    fix:
      # дерево фиксов (опционально)
    examples:
      bad: ["Неверно"]
      good: ["Верно"]
    related:
      - name: "Связанное"
        url: "/rules/related"
```

### Базовые поля

| Поле | Обязательно | Описание |
|------|-------------|----------|
| `id` | да | `category/descriptive-name`, уникальный |
| `severity` | да | 1–10, штраф в score = само значение |
| `category` | да | `cleanliness`, `readability` или произвольный |
| `name` | нет | Отображаемое название |
| `url` | нет | Ссылка на документацию |
| `suggestion` | нет | Подсказка при срабатывании |
| `detect` | да | Дерево методов обнаружения |
| `fix` | нет | Дерево автоисправления |
| `examples` | нет | Примеры bad/good |
| `related` | нет | Связанные правила |
| `enabled` | нет | Отключение правила (override) |

### Дерево детекции

Один метод или композируемое дерево с логическими операторами.

#### Листовые методы

- **`regex`** — RE2-совместимый. Без backreferences, без lookahead/lookbehind.
  ```yaml
  detect:
    regex:
      pattern: "шаблон"
      case_sensitive: false
  ```
- **`wordlist`** — список слов с учётом границ. Регистронезависимый по умолчанию.
  ```yaml
  detect:
    wordlist:
      list: ["word1", "word2"]
      case_sensitive: false
  ```
- **`contains`** — поиск подстроки.
- **`eq`** — точное совпадение.
- **`prefix`**, **`suffix`** — начало/конец текста.
- **`any`** — всегда совпадает (матчит всё, без regex). Используется как default/fallback в композитах.
- **`ref`** — делегирование detect-дереву другого правила. Разрешается в две фазы с детекцией циклов (3-цветный DFS).

#### Позиционные методы

- **`sentence_start`**, **`sentence_end`** — границы предложений (`.`, `!`, `?`)
- **`paragraph_start`**, **`paragraph_end`** — границы абзацев
- **`word_boundary`** — оборачивает child проверкой границ слова
- **`length`** — фильтр по длине совпадения (min/max)
- **`case`** — проверка регистра: `all_caps`, `all_lower`, `capitalized`, `has_upper`, `has_lower`

#### Контекстные методы

- **`before`**, **`after`** — возвращает child-совпадения с pattern в пределах max_chars до/после
- **`surrounded_by`** — совпадение между left/right маркерами
- **`position`** — первый/последний абзац
- **`threshold`** — флаг когда count ≥ N в окне (per words/paragraph/text)
- **`near`** — флаг pattern, рядом с которым есть near в окне (sentence/chars:N)
- **`exclude`** — сахар над `and` + `not`: pattern совпадения минус список других узлов. Параметры: `match` (pattern), `without` (список узлов-исключений). Для обратной совместимости принимает `list`/`words`.

#### Логические операторы

- **`and`** — пересечение. Вложенные `not` = исключения (вычитаются из пересечения)
- **`or`** — объединение. Любой child совпадает
- **`not`** — осмыслен только как исключение внутри `and`

### Дерево фиксов

Опционально. Без него правило только флаги.

#### Листовые методы

- **`replace`** — `replace: { with: "..." }`
- **`remove`** — `remove: {}`
- **`regex_replace`** — применяется только к тексту совпадения. Capture groups из detect через `$1`, `$2`
- **`when`** — условный fix: `when: { detect: ..., then: ... }`
- **`uppercase`**, **`lowercase`**, **`capitalize`**, **`sentence_case`**, **`title_case`** — трансформации регистра
- **`prepend`**, **`append`**, **`wrap`** — префикс/суффикс/обёртка
- **`trim`**, **`collapse_whitespace`** — очистка

#### Композиция

```yaml
fix:
  and:
    - lowercase: {}
    - wrap:
        prefix: "*"
        suffix: "*"
```

### Скоринг

Две оценки: чистота и читаемость (0–10). Штраф = severity каждого **pending** флага, нормализованный на 100 слов:

`score = 10 − clamp((Σseverity × 100) / word_count, 0, 10)`

- Принятые и отклонённые флаги не штрафуют
- Нормализация вшитая, без настраиваемого scale

### Загрузка правил

Три слоя с deep-merge по `id`:

1. **Base** — `RULES_DIR` (по умолч. `./rules`)
2. **Project** — `RULES_PROJECT_DIR`
3. **Override** — `RULES_OVERRIDE_DIR`

Поздний слой перекрывает ранний. Чтобы отключить правило — override с тем же `id` и `enabled: false`.

### Client vs server scope

- Клиентские правила — detect-дерево состоит только из синхронных узлов: `regex`, `wordlist`, `any`, `contains`, `eq`, `prefix`, `suffix`, `surrounded_by`, а также композитов (`and`, `or`, `not`), если все их дети синхронны. Выполняются на фронтенде мгновенно.
- Серверные правила — contain ref или «тяжёлые» стадии (LLM/NER/POS/плагин/expr с серверной функцией)
- Endpoint `/api/client-rules` отдаёт только клиентские правила
- Сервер всегда выполняет ВСЕ правила (включая клиентские) — `?full` больше не используется

### RE2 requirement

Все regex-паттерны обязаны быть RE2-совместимыми. Backreferences и lookaround отклоняются при загрузке с ошибкой. Сложный regex → правило серверное.

---

## 8. API (REST + WebSocket)

### REST endpoints

#### `GET /health` / `GET /healthz`

```json
{ "status": "ok", "plugins": ["redpolitika-ner"] }
```

#### `GET /version`

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

Локальная сборка без ldflags: `version=dev`, `commit=unknown`, `build_time=unknown`.

#### `GET /api/rules`

Все правила с метаданными (id, severity, category, detect_method, suggestion, и т.д.).

#### `GET /api/client-rules`

Только правила, исполнимые на клиенте. Формат зависит от метода детекции.

#### `POST /api/analyze`

Запрос:
```json
{ "text": "Текст для проверки" }
```

Ответ:
```json
{
  "flags": [{
    "id": "a1b2c3d4e5f6g7h8",
    "rule_id": "cleanliness/obscene",
    "category": "cleanliness",
    "severity": 8,
    "message": "...",
    "suggestion": "...",
    "auto_fix": "***",
    "anchor": { "paragraph_index": 0, "occurrence": 0, "match_text": "дурак" },
    "state": "pending",
    "rule_name": "Обсценная лексика",
    "rule_url": "/rules/obscene",
    "examples": { "bad": [...], "good": [...] },
    "related": [{"name": "...", "url": "..."}]
  }],
  "cleanliness_score": 8.5,
  "readability_score": 10.0,
  "flag_count": 1,
  "session_id": "ulid"
}
```

#### `GET /api/pages`

Список страниц документации (от плагина pages).

#### `GET /api/pages/:slug`

Контент страницы документации.

#### `GET /metrics`

Prometheus text exposition.

### WebSocket — `/ws/live`

Двунаправленный JSON-фреймовый протокол.

**Клиент → Сервер:**
| Тип | Payload | Описание |
|-----|---------|----------|
| `check` / `analyze` | `{ text, textHash }` | Запрос анализа текста (debounce 500ms) |
| `accept` | `{ flagId }` | Принять флаг |
| `reject` | `{ flagId }` | Отклонить флаг |
| `applyAll` | `{ flagIds }` | Применить все фиксы |

**Сервер → Клиент:**
| Тип | Payload | Описание |
|-----|---------|----------|
| `check_result` | `{ textHash, session_id, flags, scores }` | Результат анализа |
| `ack` | `{ action, flagId, status, error? }` | Подтверждение операции |

### Ошибки

RFC 7807 (Problem Details):

```json
{
  "type": "/errors/invalid-rule",
  "title": "Invalid rule",
  "status": 422,
  "detail": "regex pattern contains invalid syntax"
}
```

---

## 9. Плагины — gRPC-контракты и lifecycle

### Архитектура

```
┌─────────────┐     gRPC      ┌──────────────┐
│  Go ядро    │◄─────────────►│  Плагин       │
│  (Echo+FX)  │   go-plugin   │  (любой язык) │
└─────────────┘               └──────────────┘
```

### Lifecycle

1. Ядро сканирует `PLUGINS_DIR` — запускает бинарники
2. Handshake через HashiCorp go-plugin (MagicCookie)
3. gRPC connect → плагин регистрирует методы (scoped имена `plugin/method`) и функции (для expr-lang)
4. Health-check → ready
5. Graceful shutdown (отмена in-flight, таймаут 5s)
6. Hot-reload НЕ поддерживается — обновление через перезапуск ядра

### Extension points (15)

| Extension point | Назначение |
|----------------|-----------|
| `DetectFunctionProvider` | Кастомные методы детекции |
| `FixFunctionProvider` | Кастомные фиксы |
| `LLMProvider` | Провайдер LLM |
| `Migrator` | Миграции плагина |
| `FrontendBundleProvider` | JS-бандл для фронтенда |
| `BrandingProvider` | White-label / co-branding |
| `StorageProvider` | Кастомное хранилище |
| `AuthProvider` | Аутентификация |
| `RolePermissionProvider` | Роли и права |
| `BillingProvider` / `LicenseProvider` | Биллинг |
| `DocumentStorageBackend` | Хранение документов |
| `AuditLogger` | Аудит |
| `MetricsExporter` | Метрики |
| `WebhookProvider` | Webhook |
| `RuleValidator` | Валидация правил |

### Плагин-SDK

Для не-Go плагинов — отдельные репо-обёртки (`plugin-sdk-python`, `plugin-sdk-node`). Плагин реализует gRPC-сервер с go-plugin handshake.

### Имена методов

- Ядро резервирует голые имена: `regex`, `wordlist`, `pattern`, `ner`, `pos`, `llm`, `function`, `expr`
- Плагины регистрируют scoped: `plugin/method` (например `spacy/ner`)
- Голое имя разрешено, если не конфликтует с зарезервированным

---

## 10. БД и миграции

- **Один движок в runtime:** SQLite (default, zero-config) или PostgreSQL opt-in через `DB_DRIVER`
- Ядро в CE БД не использует — задел для плагинов
- Плагины пишут per-dialect SQL для обоих движков
- Миграции из коробки на старте (golang-migrate)
- Плагины — гибрид: ядро координирует версию, плагин исполняет `Migrate({dialect, dsn, targetVersion, direction})` по gRPC

---

## 11. Фронтенд

### Архитектура

- Next.js standalone SSR (`output: "standalone"`) — node server.js в production
- Caddy (:8080) — reverse proxy: `/api/*`, `/ws/*`, `/health`, `/version`, `/metrics` → Go (:8081); `/*` → Next.js (:3000)
- supervisord управляет 3 процессами: Go, Next.js, Caddy
- В dev-режиме Next.js dev server с rewrite на Go (:8080)
- CodeMirror 6 (plain text) — инлайн-подсветка флагов через decorations API
- Tailwind + shadcn/ui — UI-кит и дизайн-система
- Клиентский rule-engine — исполняет regex/wordlist правила локально
- zustand — состояние; react-query — REST-запросы; zod — валидация схем

### Merge клиентских и серверных флагов

Canonical = сервер (бэкенд пересчитывает всё). Клиент мёрджит серверные флаги по `flagId`. Конфликты по span — сервер выигрывает.

---

## 12. Флаги — ID, scoring, lifecycle

### ID флага

Флаг идентифицируется детерминированным хэшем:

```
FNV-1a 64(rule_id ‖ match_text ‖ paragraph_index ‖ occurrence_in_paragraph)
```

- **Anchor = абзац:** стабильно при правках в другом абзаце, скачет при правках в том же
- **`occurrence_in_paragraph`** — разрешает коллизию одинаковых слов в одном абзаце
- Hashids/SHA/Blake3 отвергнуты

### Lifecycle

```
raised ──→ accepted
    │          │
    ├──→ rejected
    │
    └──→ applied
```

- **Pending** — флаг активен, влияет на score
- **Accepted** — проблема решена, не штрафует
- **Rejected** — не штрафует
- **Applied** — автоисправление применено

«Применить всё» — применяет все pending флаги с настроенным автоисправлением. Конфликты пересечения span резолвятся по severity (высокий wins).

### Score

```
cleanliness = 10 − clamp((Σseverity_cleanliness_flags × 100) / word_count, 0, 10)
readability = 10 − clamp((Σseverity_readability_flags × 100) / word_count, 0, 10)
```

- Только pending флаги штрафуют
- Нормализация per-100-words, без настраиваемого scale
- Два независимых score

---

## 13. Развёртывание

### Образы

- `redpolitika-ce:local` — цельный образ (фронт + бэк + CE-плагины)
- Теги собираются через docker-compose: `docker compose build`
- CI при пуше git-тега `v*` собирает и пушит в GHCR: `ghcr.io/<owner>/redpolitika-ce:latest`, `ghcr.io/<owner>/redpolitika-ce:<version>`
- EE — отдельный образ и репо

### Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|----------|
| `PORT` | `8081` | Порт HTTP (Go-бэкенд; Caddy входной на :8080) |
| `LOG_LEVEL` | `info` | debug/info/warn/error |
| `ENVIRONMENT` | `production` | Метка окружения |
| `RULES_DIR` | `/etc/redpolitika/rules` | Базовый слой правил |
| `RULES_PROJECT_DIR` | — | Проектный слой |
| `RULES_OVERRIDE_DIR` | — | Финальный слой |
| `PLUGINS_DIR` | `/etc/redpolitika/plugins` | Бинарники плагинов |
| `DB_DRIVER` | `sqlite` | sqlite | postgres |
| `DB_DSN` | `file:/data/redpolitika.db` | Строка подключения |
| `PARAGRAPH_SEPARATOR` | — | Разделитель абзацев (пустая строка → `\n\n` в коде) |
| `PLUGIN_PAGES_FLAGS` | `--pages-dir /etc/redpolitika/pages` | Флаги для плагина pages |

### Docker Compose

```yaml
services:
  redpolitika:
    build:
      context: ..
      dockerfile: deploy/Dockerfile
    ports: ["8080:8080"]              # Caddy entry point
    environment:
      - ENVIRONMENT=production
    volumes:
      - ./rules:/etc/redpolitika/rules:ro
      - ./pages:/etc/redpolitika/pages:ro
      - redpolitika_data:/data
```

### CI-релизы

При пуше git-тега `v*` workflow собирает и пушит в GHCR:
- `ghcr.io/<owner>/redpolitika-ce:latest`
- `ghcr.io/<owner>/redpolitika-ce:<version>`

---

## 14. Лицензирование

**CE — Business Source License 1.1** (HashiCorp-шаблон).

- **Change Date:** 2030-07-18 → Apache 2.0
- **Additional Use Grant:** свободное использование для:
  - Физических лиц
  - Юрлиц с годовой выручкой < 15 млн RUB (РФ) / < $400K (остальной мир)
- Два независимых фиксированных порога — без привязки курсом
- Обязательно покрытие аффилированных лиц (защита от дробления)

**EE-плагины** — проприетарная лицензия, отдельный репо/образ (Mattermost-модель). CE-ядро проектируется так, чтобы EE надстраивался через extension points без переписывания.

---

*Документ описывает CE-ядро. EE-функции и соответствующие extension points описаны в отдельной документации.*
