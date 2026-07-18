# Redpolitika

**Open-core text checking service against editorial policy rules in YAML.**  
Think "Glavred, but for any editorial policy."

Define your editorial style as YAML rules — `Redpolitika` checks text, flags violations, and suggests fixes. Works out of the box with Docker Compose.

---

## Features

- **YAML-defined rules** — composable detect/fix method trees
- **Client + server** — regex/wordlist rules run locally, composite rules on server
- **WebSocket live analysis** — real-time flags as you type
- **CodeMirror 6 editor** — inline flag display and navigation
- **Extension points** — gRPC plugin system for custom providers
- **Scoring** — cleanliness + readability, normalized per 100 words

## Quick start

```bash
docker compose -f deploy/docker-compose.yml up
```

Open `http://localhost:8080`. Paste text, see flags.

## Architecture

```
backend/        Go (Echo, Uber FX, DDD/ports-and-adapters)
frontend/       Next.js static export (served by Go)
ce-plugins/     Reference plugin examples
plugin-sdk/     SDK for non-Go plugins
deploy/         Dockerfile + docker-compose.yml
```

Stack: Go backend, Next.js static frontend, SQLite/Postgres, CodeMirror 6, gRPC plugins, tailwind + shadcn/ui.

## Rules

Define rules in YAML with composable detect/fix method trees:

```yaml
rules:
  - id: typography/ellipsis
    severity: 5
    category: typography
    detect:
      regex: …
    fix:
      replace:
        with: …
    suggestion: Замените три точки на символ многоточия
```

Rule layer system: `base → project → override` with deep merge by `id`.

## Documentation

- [Rules reference](docs/rules.md) — YAML rule format, detect/fix trees, examples
- [Deployment guide](docs/deployment.md) — Docker, config, environment variables
- [API reference](docs/api.md) — REST + WebSocket endpoints
- [AI Agent Skill](docs/ai-agent-skill.md) — writing rules with AI agents

## License

**Business Source License 1.1** — see [LICENSE](LICENSE).

- **Change Date:** 2030-07-18 → Apache 2.0
- **Additional Use Grant:** Free for non-production use and small entities:
  - ≤ 15M RUB annual revenue (Russian Federation entities)
  - ≤ $400K annual revenue (entities outside Russia)
  - Two independent fixed thresholds — no currency rate linkage

Enterprise features (EE) are proprietary and separate.

## Notice

See [NOTICE](NOTICE) for third-party licenses.
