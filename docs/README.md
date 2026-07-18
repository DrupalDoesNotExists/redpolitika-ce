# redpolitika documentation

| Document | Description |
|----------|-------------|
| [Rules](rules.md) | YAML rule format — detect/fix method trees, examples |
| [Cookbook](cookbook.md) | Composite patterns (whitelist, threshold, near, …) |
| [Schema](schema.json) | JSON Schema for rule files (editor validation) |
| [Deployment](deployment.md) | Docker Compose, configuration, environment variables |
| [API](api.md) | REST + WebSocket endpoints, OpenAPI spec |
| [AI Agent Skill](ai-agent-skill.md) | Writing rules with AI coding agents |

Quick start: `docker compose -f deploy/docker-compose.yml up` → http://localhost:8080

Image has no bundled rules — mount your YAML and set `RULES_DIR` (see [deployment](deployment.md)).
