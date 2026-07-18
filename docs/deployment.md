# Deployment

## Docker Compose (recommended)

```bash
docker compose -f deploy/docker-compose.yml up
```

Open `http://localhost:8080`.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `ENVIRONMENT` | `development` | Environment label |
| `DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `DB_DSN` | `file:redpolitika.db?cache=shared&_journal_mode=WAL` | DB connection string |
| `RULES_DIR` | `./rules` | Base rules directory |
| `RULES_PROJECT_DIR` | — | Project rules override layer |
| `RULES_OVERRIDE_DIR` | — | Per-environment override layer |
| `PLUGINS_DIR` | — | Plugin binaries directory |
| `STATIC_DIR` | `../frontend/out` | Frontend static files |
| `PARAGRAPH_SEPARATOR` | `\n\n` | Paragraph separator |

### PostgreSQL

```bash
DB_DRIVER=postgres DB_DSN="postgres://user:password@localhost:5432/redpolitika?sslmode=disable"
```

Migrations apply automatically on first run.

## Building from source

### Backend (Go 1.22+)

```bash
cd backend
go build -o redpolitika ./cmd/api/
./redpolitika
```

### Frontend

```bash
cd frontend
npm install
npm run build     # → frontend/out/ (static export)
```

Production: set `STATIC_DIR` to the build output directory.

### Combined

```bash
cd frontend && npm install && npm run build
cd ../backend && go build -o redpolitika ./cmd/api/
STATIC_DIR=../frontend/out ./redpolitika
```

## Docker

```bash
docker build -f deploy/Dockerfile -t redpolitika:latest .
docker run -p 8080:8080 redpolitika:latest
```

## Configuration example

```bash
export PORT=8080
export DB_DRIVER=sqlite
export RULES_DIR=/etc/redpolitika/rules
export RULES_PROJECT_DIR=/etc/redpolitika/project
export STATIC_DIR=/var/www/redpolitika
```

## Related

- [Rules reference](rules.md)
- [API reference](api.md)
