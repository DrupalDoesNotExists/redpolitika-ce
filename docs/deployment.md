# Deployment

## Run from Docker Hub (out of the box)

```bash
docker run --rm -p 8080:8080 \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  YOUR_DOCKERHUB_USER/redpolitika-ce:latest
```

Open `http://localhost:8080`.

The image includes the UI and API. Rules are **not** bundled — mount a directory
with your YAML into `/etc/redpolitika/rules` (or leave it empty: the server starts,
flags stay empty until you add rules).

### Persist SQLite

```bash
docker run --rm -p 8080:8080 \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  -v redpolitika-data:/data \
  YOUR_DOCKERHUB_USER/redpolitika-ce:latest
```

## Docker Compose (build locally)

```bash
mkdir -p deploy/rules deploy/plugins
docker compose -f deploy/docker-compose.yml up --build
```

Open `http://localhost:8080`.

## Environment variables

All have defaults inside the image — you only override what you need.

| Variable | Image default | Description |
|----------|---------------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `ENVIRONMENT` | `production` | Label only |
| `STATIC_DIR` | `/frontend/out` | Built Next.js export (baked into image) |
| `RULES_DIR` | `/etc/redpolitika/rules` | Base YAML rules (mount your pack here) |
| `RULES_PROJECT_DIR` | — | Project override layer |
| `RULES_OVERRIDE_DIR` | — | Per-env override layer |
| `PLUGINS_DIR` | `/etc/redpolitika/plugins` | Plugin binaries |
| `DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `DB_DSN` | `file:/data/redpolitika.db?…` | DB connection string |
| `PARAGRAPH_SEPARATOR` | `\n\n` | Paragraph split |

### PostgreSQL

```bash
docker run --rm -p 8080:8080 \
  -e DB_DRIVER=postgres \
  -e DB_DSN="postgres://user:password@db:5432/redpolitika?sslmode=disable" \
  -v "$PWD/rules:/etc/redpolitika/rules:ro" \
  YOUR_DOCKERHUB_USER/redpolitika-ce:latest
```

Migrations apply automatically on first run.

## Building from source

### Backend (Go)

```bash
cd backend
go build -o redpolitika ./cmd/api/
RULES_DIR=./rules STATIC_DIR=../frontend/out ./redpolitika
```

### Frontend

```bash
cd frontend
npm install
npm run build     # → frontend/out/
```

### Combined image

```bash
docker build -f deploy/Dockerfile -t redpolitika-ce:local .
docker run --rm -p 8080:8080 \
  -v "$PWD/my-rules:/etc/redpolitika/rules:ro" \
  redpolitika-ce:local
```

## Docker Hub (CI release)

On git tag `v*`, workflow `.github/workflows/release.yml` builds and pushes:

- `{DOCKERHUB_USERNAME}/redpolitika-ce:latest`
- `{DOCKERHUB_USERNAME}/redpolitika-ce:<version>`
- `{DOCKERHUB_USERNAME}/redpolitika-ce:full`

Repo secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`.

## Related

- [Rules reference](rules.md)
- [Cookbook](cookbook.md)
- [API reference](api.md)
