# AGENTS.md — redpolitika

redpolitika — open-core сервис проверки текста по правилам редполитики в YAML. License BSL (CE) + proprietary EE plugins.

## Stack
- **Backend:** Go, Echo, Uber FX (DI), DDD/ports-and-adapters, zap
- **Frontend:** Next.js (standalone SSR, Caddy reverse proxy), CodeMirror 6, Tailwind + shadcn/ui
- **Database:** SQLite (default) / PostgreSQL opt-in (one engine at runtime)
- **Plugins:** HashiCorp go-plugin, gRPC, multi-language
- **Rules:** YAML on disk, deep-merge layers base→project→override by id

## Architecture invariants
- **DDD discipline:** rich domain (entity methods, not anemic), use-cases, Value Objects, validation in constructors. Domain knows nothing about infrastructure.
- **Error encapsulation:** each layer wraps lower-layer errors in its own. Only application errors reach outside.
- **Plugin architecture:** kernel knows extension points (interfaces), not plugin classes.
- **Regex:** RE2-subset only. Backreferences/lookaround → load error.
- **Flag ID:** FNV-1a 64 from `rule_id + match_text + paragraph_index + occurrence_in_paragraph`.
- **Severity:** int 1–10. Two scores (cleanliness/readability), normalized per 100 words, no configurable scale.
- **License:** BSL 1.1 (change date → Apache 2.0), two independent fixed thresholds.

## Monorepo structure
```
backend/        Go core (cmd/api/, internal/{domain,usecase,infra,transport}, proto/)
frontend/       Next.js standalone SSR
ce-plugins/     Reference CE plugins
plugin-sdk/     SDK for non-Go plugins
deploy/         Dockerfile + docker-compose.yml
docs/           Documentation + SPEC.md
```

## Reference
Full specification: `docs/SPEC.md`
