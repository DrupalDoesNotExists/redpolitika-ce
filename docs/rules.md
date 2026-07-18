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

Check character case of the child match (or `\S+` words if no child).

```yaml
detect:
  case:
    mode: "all_caps"   # all_caps | all_lower | capitalized | has_upper | has_lower
    child:
      wordlist:
        list: ["москва"]
```

### Contextual

#### `before`, `after`

Return **child** matches that have `pattern` within `max_chars` before/after them.
Without `pattern`, falls back to one character before/after each child match.

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

First/last paragraph (`type`) with optional child, or numeric `from`/`to` (compat).

```yaml
detect:
  position:
    type: "first_paragraph"   # "first_paragraph" | "last_paragraph"
    child:
      wordlist:
        list: ["введение"]
```

#### `threshold`

Flag inner matches only when count ≥ `count` per window.

```yaml
detect:
  threshold:
    count: 3
    per: words          # words | paragraph | text
    window: 100         # for per=words
    child:
      wordlist:
        list: ["вроде"]
```

#### `near`

Flag `pattern` matches that have a second pattern nearby.

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

Sugar over `and` + `not` + `wordlist`: keep child matches except whitelist words.

```yaml
detect:
  exclude:
    list: ["ёлка", "ёжик"]
    case_sensitive: false
    child:
      regex: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*"
```

Equivalent composite:

```yaml
detect:
  and:
    - regex: "[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*"
    - not:
        wordlist:
          list: ["ёлка", "ёжик"]
```

### Logical operators

#### `and`

Intersection — all children must match. Nested `not` children act as **exclusions**
(subtracted from the intersection), not as standalone negation.

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

**Only meaningful as an exclusion inside `and`.** Standalone `not` returns its
child matches unchanged (does not invert the document).

```yaml
# Correct — exclude whitelist inside and:
detect:
  and:
    - wordlist: { list: ["ещё", "всё"] }
    - not:
        wordlist: { list: ["всё"] }

# Standalone not — returns child matches (no inversion):
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

## Inline suppressions

Disable rules for a span of the document (HTML comments, stripped from scoring only):

```text
<!-- rp:disable redundancy/very -->
это очень важно
<!-- rp:enable -->

одна строка <!-- rp:disable-line redundancy/very -->
```

Omit `rule-id` to suppress all rules in the range. `rp:disable-line` covers the whole line.

---

## Fix tree

Optional. Without it, rule flags only.

**Limitation (B1):** every fix sees the match string plus a context
`(text, start, end, groups)` — the paragraph text, match span, and detect
capture groups. Fixes still **replace only the match span** in the document;
they cannot expand/shrink the edit beyond that span via the fix tree alone.

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

Applies to the **match text only**, never to the whole document.
If the detect `regex` had capture groups, `$1` / `$2` / `${1}` in `replacement`
are filled from those detect groups (E4). Otherwise `pattern` runs on the match.

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

Apply nested fix only if a detect condition matches the match string.

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
  title_case: {}       # each word title-cased (Unicode-aware)
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

## Scoring

Two scores: cleanliness and readability (0–10). Penalty = severity of each
**pending** (raised) flag, normalised per 100 words.

**Accept semantics:** accepting a flag means the problem is resolved for scoring —
accepted flags do **not** reduce the score (same as rejected). Only pending flags penalise.

---

## Rule loading

redpolitika loads rules from directories you provide (nothing is bundled in the image).
Three layers (deep-merge by `id`):

1. **Base** — `RULES_DIR` (default `./rules`)
2. **Project** — `RULES_PROJECT_DIR`
3. **Override** — `RULES_OVERRIDE_DIR`

To disable a rule, provide override with same `id` and no `detect`.

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
