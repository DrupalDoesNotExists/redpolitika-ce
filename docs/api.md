# API Reference

Base URL: `http://localhost:8080`

---

## REST endpoints

### `GET /health`

```json
{
  "status": "ok"
}
```

### `GET /version`

```json
{
  "version": "0.1.0-dev",
  "module": "ce",
  "component": "redpolitika"
}
```

### `GET /api/rules`

All loaded rules with metadata.

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

Client-side rules only — regex/wordlist leaves the frontend can apply locally.

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

Fields vary by detect method:
- `method: "regex"` — includes `pattern` (string), `engine: "re2"`
- `method: "wordlist"` — includes `words` (array), `case_sensitive` (bool)

### `POST /api/analyze`

Analyze text.

Query params:
- `?full=true` — include client-side rules (default: server-side only)

Request:

```json
{
  "text": "Ты дурак. Это очень важно."
}
```

Response:

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

Bidirectional JSON-framed protocol.

### Client → Server

#### check/analyze

```json
{"type": "check", "text": "Text to analyze", "textHash": "", "full": false}
```

- `type: "check"` and `type: "analyze"` are identical in behavior
- Server debounces 500ms

#### accept/reject

```json
{"type": "accept", "flagId": "a1b2c3d4e5f6g7h8"}
{"type": "reject", "flagId": "a1b2c3d4e5f6g7h8"}
```

#### applyAll

```json
{"type": "applyAll", "flagIds": ["a1b2c3d4e5f6g7h8", "i9j0k1l2m3n4o5p6"]}
```

### Server → Client

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

## Error format

Errors follow RFC 7807 (Problem Details):

```json
{
  "type": "https://redpolitika.dev/errors/invalid-rule",
  "title": "Invalid rule",
  "status": 422,
  "detail": "regex pattern contains invalid syntax"
}
```

---

## Scores

- **Cleanliness** — 0–10, penalized by `category: "cleanliness"` flags
- **Readability** — 0–10, penalized by `category: "readability"` flags
- Normalized per 100 words
- Each flag deducts its severity value

---

## OpenAPI spec

```yaml
openapi: "3.1.0"
info:
  title: redpolitika API
  version: 0.1.0
  description: "Text checking against editorial policy rules"
paths:
  /health:
    get:
      summary: Health check
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: "ok"
  /version:
    get:
      summary: Version info
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  version:
                    type: string
                    example: "0.1.0-dev"
                  module:
                    type: string
                    example: "ce"
                  component:
                    type: string
                    example: "redpolitika"
  /api/rules:
    get:
      summary: List all rules
      responses:
        "200":
          description: Rule list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/RuleMeta"
  /api/client-rules:
    get:
      summary: List client-side rules
      responses:
        "200":
          description: Client rule list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/ClientRule"
  /api/analyze:
    post:
      summary: Analyze text
      parameters:
        - name: full
          in: query
          schema:
            type: boolean
          description: Include client-side rules
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                text:
                  type: string
                  example: "Ты дурак"
      responses:
        "200":
          description: Analysis result
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AnalysisResult"
components:
  schemas:
    RuleMeta:
      type: object
      properties:
        id: { type: string }
        severity: { type: integer }
        category: { type: string }
        detect_method: { type: string }
        suggestion: { type: string }
        auto_fix: { type: string, nullable: true }
        client_side: { type: boolean }
        name: { type: string }
        url: { type: string }
        examples:
          type: object
          properties:
            bad: { type: array, items: { type: string } }
            good: { type: array, items: { type: string } }
        related:
          type: array
          items:
            type: object
            properties:
              name: { type: string }
              url: { type: string }
    ClientRule:
      type: object
      properties:
        id: { type: string }
        severity: { type: integer }
        category: { type: string }
        method: { type: string }
        pattern: { type: string }
        words: { type: array, items: { type: string } }
        case_sensitive: { type: boolean }
        suggestion: { type: string }
        auto_fix: { type: string, nullable: true }
        engine: { type: string }
        name: { type: string }
        url: { type: string }
        examples:
          type: object
          properties:
            bad: { type: array, items: { type: string } }
            good: { type: array, items: { type: string } }
        related:
          type: array
          items:
            type: object
            properties:
              name: { type: string }
              url: { type: string }
    AnalysisResult:
      type: object
      properties:
        flags:
          type: array
          items:
            $ref: "#/components/schemas/Flag"
        cleanliness_score: { type: number }
        readability_score: { type: number }
        flag_count: { type: integer }
        session_id: { type: string }
    Flag:
      type: object
      properties:
        id: { type: string }
        rule_id: { type: string }
        category: { type: string }
        severity: { type: integer }
        message: { type: string }
        suggestion: { type: string }
        auto_fix: { type: string, nullable: true }
        anchor:
          type: object
          properties:
            paragraph_index: { type: integer }
            occurrence: { type: integer }
            match_text: { type: string }
        state: { type: string }
        rule_name: { type: string }
        rule_url: { type: string }
        examples:
          type: object
          properties:
            bad: { type: array, items: { type: string } }
            good: { type: array, items: { type: string } }
        related:
          type: array
          items:
            type: object
            properties:
              name: { type: string }
              url: { type: string }
```
