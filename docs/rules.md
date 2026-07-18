# Rules

Rules are YAML files on disk. redpolitika loads, validates, and compiles them at startup. Invalid rules block startup — no silent skips.

---

## File format

```yaml
rules:
  - id: "category/name"
    severity: 5
    category: "cleanliness"
    name: "Human readable name"
    url: "/rules/name"
    suggestion: "Что делать"
    detect:
      # method tree
    fix:
      # fix tree (optional)
    examples:
      bad:
        - "Неверно"
      good:
        - "Верно"
    related:
      - name: "Related"
        url: "/rules/related"
```

### Base fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | `category/descriptive-name`, unique |
| `severity` | yes | 1–10, higher = worse |
| `category` | yes | `cleanliness`, `readability`, or custom string |
| `name` | no | Display name |
| `url` | no | Link to docs |
| `suggestion` | no | Shown to user when flag fires |
| `examples` | no | Before/after examples |
| `related` | no | Related rule references |

---

## Detection tree

Single method or composable tree with logical operators.

### Leaf methods

#### `regex`

RE2-compatible. No backreferences, no lookahead/lookbehind.

```yaml
detect:
  regex: "pattern"
```

#### `wordlist`

Word-boundary aware, case-insensitive by default.

```yaml
detect:
  wordlist:
    list: ["word1", "word2"]
    case_sensitive: false
```

#### `contains`

Substring match, no word boundaries.

```yaml
detect:
  contains:
    value: "substring"
    case_sensitive: false
```

#### `eq`

Exact string match.

```yaml
detect:
  eq: "exact text"
```

#### `prefix`, `suffix`

Match at start/end of text.

```yaml
detect:
  prefix: "Start"
```

```yaml
detect:
  suffix: "End"
```

### Positional

#### `sentence_start`, `sentence_end`

Bound by `.`, `!`, `?` sentence boundaries.

```yaml
detect:
  sentence_start:
    regex: "^[а-я]"
```

#### `paragraph_start`, `paragraph_end`

Bound by paragraph boundaries.

```yaml
detect:
  paragraph_start:
    wordlist:
      list: ["Во-первых"]
```

#### `word_boundary`

Wrap child with word-boundary check.

```yaml
detect:
  word_boundary:
    contains:
      value: "word"
```

#### `length`

Filter by match length.

```yaml
detect:
  length:
    min: 3
    max: 50
    child:
      regex: "\\b\\w+\\b"
```

#### `case`

Check character case of child match.

```yaml
detect:
  case:
    mode: "upper"   # "upper" | "lower" | "title"
    child:
      wordlist:
        list: ["москва"]
```

### Contextual

#### `before`, `after`

Match must be preceded/followed by pattern within distance.

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

Match must be between left/right markers.

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

First or last paragraph only.

```yaml
detect:
  position:
    type: "first_paragraph"   # "first_paragraph" | "last_paragraph"
    child:
      wordlist:
        list: ["введение"]
```

### Logical operators

#### `and`

Intersection — all children must match.

```yaml
detect:
  and:
    - regex: "очень"
    - contains:
        value: "важно"
```

#### `or`

Union — any child matches.

```yaml
detect:
  or:
    - regex: "кр[ао]сивый"
    - wordlist:
        list: ["прелестный"]
```

#### `not`

Negate child.

```yaml
detect:
  not:
    contains:
      value: "не"
```

### Composition examples

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

## Fix tree

Optional. Without it, rule flags only.

### Leaf methods

#### `replace`

```yaml
fix:
  replace:
    with: "replacement"
```

#### `remove`

```yaml
fix:
  remove: {}
```

#### `regex_replace`

```yaml
fix:
  regex_replace:
    pattern: "(\\w+) (\\w+)"
    replacement: "$2, $1"
```

#### Case transforms

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
  capitalize: {}       # first letter uppercase
```
```yaml
fix:
  sentence_case: {}    # first upper, rest lower
```
```yaml
fix:
  title_case: {}       # each word uppercase
```

#### Text manipulation

```yaml
fix:
  prepend:
    with: "prefix "
```
```yaml
fix:
  append:
    with: " suffix"
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

### Composition

```yaml
fix:
  and:
    - lowercase: {}
    - wrap:
        prefix: "*"
        suffix: "*"
```

---

## Rule loading

redpolitika loads rules from directories (default: `./rules`). Supports three layers:

1. **Base** — shipped defaults
2. **Project** — project overrides
3. **Override** — per-env overrides

Set via environment variables:
- `RULES_DIR` — base layer
- `RULES_PROJECT_DIR` — project layer
- `RULES_OVERRIDE_DIR` — override layer

Layers are deep-merged by `id`. To disable a rule, provide override with same `id` and no `detect`.

---

## RE2 requirement

All regex patterns must be RE2-compatible. Backreferences and lookaround are rejected at load time.

---

## Client vs server scope

Client-side rules (regex/wordlist leaves) are sent to frontend for instant local checking. Composite tree rules run server-side.

---

## Examples

### Typography

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

### Redundancy

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

### Wordlist

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

### Tautology

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

### Composition — lowercase after colon

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
