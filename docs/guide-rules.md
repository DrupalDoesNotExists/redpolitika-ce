---
title: Правила
description: Полный формат YAML-правил — дерево detect/fix, все методы, примеры
weight: 30
lang: ru
---

# Правила

Правила — YAML-файлы на диске. Redpolitika загружает, валидирует и компилирует их при старте. Некорректные правила блокируют запуск — без silent skip.

---

## Формат файла

```yaml
rules:
  - id: "category/name"
    severity: 5
    category: "cleanliness"
    name: "Человеческое название"
    url: "/rules/name"
    suggestion: "Что делать"
    detect:
      # дерево методов
    fix:
      # дерево фиксов (опционально)
    examples:
      bad:
        - "Неверно"
      good:
        - "Верно"
    related:
      - name: "Связанное правило"
        url: "/rules/related"
```

### Базовые поля

| Поле | Обязательно | Описание |
|------|-------------|----------|
| `id` | да | `category/descriptive-name`, уникальный |
| `severity` | да | 1–10, выше = хуже |
| `category` | да | `cleanliness`, `readability` или произвольный |
| `name` | нет | Отображаемое название |
| `url` | нет | Ссылка на документацию правила |
| `suggestion` | нет | Подсказка пользователю при срабатывании |
| `examples` | нет | Примеры до/после |
| `related` | нет | Ссылки на связанные правила |

---

## Дерево детекции

Один метод или композируемое дерево с логическими операторами.

### Листовые методы

#### `regex`

RE2-совместимый. Без backreferences, без lookahead/lookbehind.

```yaml
detect:
  regex: "pattern"
```

С флагами:

```yaml
detect:
  regex:
    pattern: "шаблон"
    case_sensitive: true
```

#### `wordlist`

Поиск по списку слов с учётом границ слов. По умолчанию регистронезависимый.

```yaml
detect:
  wordlist:
    list: ["word1", "word2"]
    case_sensitive: false
```

#### `contains`

Поиск подстроки без учёта границ слов.

```yaml
detect:
  contains:
    value: "подстрока"
    case_sensitive: false
```

#### `eq`

Точное совпадение строки.

```yaml
detect:
  eq: "точный текст"
```

#### `prefix`, `suffix`

Совпадение в начале/конце текста.

```yaml
detect:
  prefix: "Начало"
```

```yaml
detect:
  suffix: "Конец"
```

### Позиционные

#### `sentence_start`, `sentence_end`

Границы предложений (`.`, `!`, `?`). С дочерним методом — оставляет совпадения на границе предложения, без `^`/`$` в regex.

```yaml
detect:
  sentence_start:
    regex:
      pattern: "[а-яё]"
      case_sensitive: true
```

#### `paragraph_start`, `paragraph_end`

Границы абзацев. С дочерним методом — оставляет совпадения на границе.

```yaml
detect:
  paragraph_start:
    wordlist:
      list: ["Во-первых"]
```

#### `word_boundary`

Оборачивает дочерний метод проверкой границ слова.

```yaml
detect:
  word_boundary:
    contains:
      value: "слово"
```

#### `length`

Фильтр по длине совпадения.

```yaml
detect:
  length:
    min: 3
    max: 50
    child:
      regex: "\\b\\w+\\b"
```

#### `case`

Проверка регистра символов совпадения (или `\S+` слов, если без child).

```yaml
detect:
  case:
    mode: "all_caps"   # all_caps | all_lower | capitalized | has_upper | has_lower
    child:
      wordlist:
        list: ["москва"]
```

### Контекстные

#### `before`, `after`

Возвращает совпадения **child**, у которых есть `pattern` в пределах `max_chars` до/после. Без `pattern` — проверяет один символ до/после каждого совпадения.

```yaml
detect:
  before:
    pattern:
      regex: "уважаемый\\s+"
    max_chars: 50
    child:
      wordlist:
        list: ["господин"]
```

#### `surrounded_by`

Совпадение должно быть между левым и правым маркерами.

```yaml
detect:
  surrounded_by:
    left: "«"
    right: "»"
    child:
      contains:
        value: "важно"
```

#### `position`

Первый/последний абзац с опциональным child, или числовые `from`/`to`.

```yaml
detect:
  position:
    type: "first_paragraph"   # first_paragraph | last_paragraph
    child:
      wordlist:
        list: ["введение"]
```

#### `threshold`

Флаг внутренних совпадений только когда их количество ≥ `count` в окне.

```yaml
detect:
  threshold:
    count: 3
    per: words          # words | paragraph | text
    window: 100         # для per=words
    child:
      wordlist:
        list: ["вроде"]
```

#### `near`

Флаг совпадений `pattern`, рядом с которыми есть `near` в окне.

```yaml
detect:
  near:
    pattern:
      wordlist: { list: ["X"] }
    near:
      wordlist: { list: ["Y"] }
    window: sentence    # sentence | chars:N | integer chars
```

#### `exclude`

Сахар над `and` + `not` + `wordlist`: исключает слова из списка.

```yaml
detect:
  exclude:
    list: ["ёлка", "ёжик"]
    case_sensitive: false
    child:
      regex: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*"
```

Эквивалент:

```yaml
detect:
  and:
    - regex: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*"
    - not:
        wordlist:
          list: ["ёлка", "ёжик"]
```

### Логические операторы

#### `and`

Пересечение — все дочерние методы должны совпасть. Вложенные `not` работают как **исключения** (вычитаются из пересечения).

```yaml
detect:
  and:
    - regex: "очень"
    - contains:
        value: "важно"
```

#### `or`

Объединение — любой дочерний метод совпадает.

```yaml
detect:
  or:
    - regex: "кр[ао]сивый"
    - wordlist:
        list: ["прелестный"]
```

#### `not`

**Осмыслен только как исключение внутри `and`.** Отдельный `not` возвращает совпадения дочернего метода без изменений (не инвертирует документ).

```yaml
# Корректно — исключение внутри and:
detect:
  and:
    - wordlist: { list: ["ещё", "всё"] }
    - not:
        wordlist: { list: ["всё"] }

# Отдельный not — возвращает child без изменений:
detect:
  not:
    contains:
      value: "не"
```

### Пример композиции

```yaml
detect:
  and:
    - sentence_start: {}
    - not:
        wordlist:
          list: ["Заглавная"]
    - length:
        min: 1
        max: 3
        child:
          regex: "\\b\\w+\\b"
```

---

## Inline-подавление

Отключение правил на участке текста через HTML-комментарии (исключаются из скоринга):

```text
<!-- rp:disable redundancy/very -->
это очень важно
<!-- rp:enable -->

одна строка <!-- rp:disable-line redundancy/very -->
```

Без `rule-id` подавляются все правила. `rp:disable-line` действует на всю строку.

---

## Дерево фиксов

Опционально. Без него правило только флаги, без автоправки.

**Ограничение:** каждый фикс видит строку совпадения + контекст `(text, start, end, groups)` — текст абзаца, позиции совпадения и capture groups из детекции. Фикс заменяет **только span совпадения**, не расширяя его.

### Листовые методы

#### `replace`

```yaml
fix:
  replace:
    with: "замена"
```

#### `remove`

```yaml
fix:
  remove: {}
```

#### `regex_replace`

Применяется **только к тексту совпадения**, не ко всему документу. Если у detect `regex` были capture groups, `$1` / `$2` / `${1}` в `replacement` заполняются из них. Иначе `pattern` применяется к тексту совпадения.

```yaml
detect:
  regex: "(у нас) (есть)"
fix:
  regex_replace:
    pattern: ".*"
    replacement: "$2, $1"
# → "есть, у нас"
```

#### `when`

Применить вложенный фикс только если условие детекции совпадает с текстом совпадения.

```yaml
fix:
  when:
    detect:
      case: { mode: all_lower }
    then:
      regex_replace:
        pattern: "ё"
        replacement: "е"
```

#### Трансформации регистра

```yaml
fix:
  uppercase: {}
```

```yaml
fix:
  lowercase: {}
```

```yaml
fix:
  capitalize: {}       # первая буква заглавная
```

```yaml
fix:
  sentence_case: {}    # первая заглавная, остальные строчные
```

```yaml
fix:
  title_case: {}       # каждое слово с заглавной (Unicode-aware)
```

#### Манипуляции с текстом

```yaml
fix:
  prepend:
    with: "префикс "
```

```yaml
fix:
  append:
    with: " суффикс"
```

```yaml
fix:
  wrap:
    prefix: "«"
    suffix: "»"
```

```yaml
fix:
  trim: {}
```

```yaml
fix:
  collapse_whitespace: {}
```

### Композиция фиксов

```yaml
fix:
  and:
    - lowercase: {}
    - wrap:
        prefix: "*"
        suffix: "*"
```

---

## Скоринг

Две оценки: чистота и читаемость (0–10). Штраф = severity каждого **pending** (непринятого) флага, нормализованный на 100 слов.

**Принятие флага:** флаг считается решённым для скоринга — принятые флаги **не** снижают оценку (как и отклонённые). Только pending флаги штрафуют.

---

## Загрузка правил

Redpolitika загружает правила из директорий (ничего не встроено в образ). Три слоя (deep-merge по `id`):

1. **Base** — `RULES_DIR` (по умолч. `./rules`)
2. **Project** — `RULES_PROJECT_DIR`
3. **Override** — `RULES_OVERRIDE_DIR`

Чтобы отключить правило, дайте override с тем же `id` без `detect`.

---

## Требование RE2

Все regex-паттерны должны быть RE2-совместимы. Backreferences и lookaround отклоняются при загрузке.

---

## Client vs server scope

Client-side правила (regex/wordlist листья) отправляются фронтенду для мгновенной локальной проверки. Композитные древовидные правила выполняются на сервере.

---

## Полные примеры

### Типография

```yaml
rules:
  - id: "typography/ellipsis"
    severity: 5
    category: "readability"
    detect:
      regex: "\\.\\.\\."
    fix:
      replace:
        with: "…"
    suggestion: "Замените три точки на символ многоточия"
```

### Избыточность

```yaml
rules:
  - id: "redundancy/very"
    severity: 5
    category: "cleanliness"
    detect:
      regex: "очень\\s+(важно|необходимо|нужно)"
    fix:
      lowercase: {}
    suggestion: "Уберите усилитель «очень»"
```

### Список слов

```yaml
rules:
  - id: "cleanliness/obscene"
    severity: 8
    category: "cleanliness"
    detect:
      wordlist:
        list: ["дурак", "идиот", "кретин"]
        case_sensitive: false
    fix:
      replace:
        with: "***"
```

### Тавтология

```yaml
rules:
  - id: "readability/tautology"
    severity: 3
    category: "readability"
    detect:
      or:
        - eq: "масло масляное"
        - eq: "неожиданный сюрприз"
    fix:
      remove: {}
```

### Композиция — строчная после двоеточия

```yaml
rules:
  - id: "capitalization/after-colon"
    severity: 4
    category: "cleanliness"
    detect:
      after:
        pattern:
          contains:
            value: ": "
        max_chars: 80
        child:
          regex: "^[А-Я]"
    suggestion: "После двоеточия — строчная буква"
```

---

## Связанное

- [Рецепты](cookbook.md) — готовые паттерны
- [API](guide-api.md) — программная проверка текста
- [Развёртывание](guide-deployment.md) — настройка RULES_DIR
