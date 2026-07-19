# Frontend builder
FROM node:24-alpine AS frontend
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Go builder
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache protoc
WORKDIR /build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
ARG LICENSE=BSL-1.1
ENV VERSION=$VERSION COMMIT=$COMMIT BUILD_TIME=$BUILD_TIME LICENSE=$LICENSE
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Generate protobuf Go code from .proto files (A27 extension points)
RUN for dir in proto/detect proto/fix proto/identity proto/llm proto/migrator proto/pages; do \
		ls "$$dir"/*.proto >/dev/null 2>&1 && cd "$$dir" && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative *.proto && cd /build || true; \
	done
RUN CGO_ENABLED=0 go build -ldflags="-s -w \
	-X github.com/drupaldoesnotexists/redpolitika/ce/internal/buildinfo.Version=${VERSION} \
	-X github.com/drupaldoesnotexists/redpolitika/ce/internal/buildinfo.Commit=${COMMIT} \
	-X github.com/drupaldoesnotexists/redpolitika/ce/internal/buildinfo.BuildTime=${BUILD_TIME} \
	-X github.com/drupaldoesnotexists/redpolitika/ce/internal/buildinfo.License=${LICENSE}" \
	-o /redpolitika ./cmd/api

# Plugin builder — standalone Go modules for CE plugins
FROM golang:1.26-alpine AS plugin-builder
RUN apk add --no-cache protoc
WORKDIR /build
COPY ce-plugins/pages/go.mod ce-plugins/pages/go.sum ./
RUN go mod download
COPY ce-plugins/pages/ ./
# Generate .pb.go from plugin-local .proto copies
RUN protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/identity/identity.proto proto/pages/pages.proto && \
    CGO_ENABLED=0 go build -o /pages-plugin .

# Runtime — Caddy ingress + Go API + Next.js
FROM caddy:alpine
RUN apk add --no-cache ca-certificates tzdata nodejs
COPY --from=builder /redpolitika /usr/local/bin/redpolitika
# CE plugins
COPY --from=plugin-builder /pages-plugin /etc/redpolitika/plugins/pages

# Next.js standalone output
COPY --from=frontend /frontend/.next/standalone /nextjs
COPY --from=frontend /frontend/public /nextjs/public/
COPY --from=frontend /frontend/.next/static /nextjs/.next/static

# Caddy + entrypoint
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose Caddy port
EXPOSE 8080
VOLUME ["/data", "/etc/redpolitika/rules", "/etc/redpolitika/plugins"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- "http://127.0.0.1:8080/health" || exit 1

# ENVs for Go API (PORT set by entrypoint)
ENV LOG_LEVEL=info \
	ENVIRONMENT=production \
	STATIC_DIR=/frontend/out \
	RULES_DIR=/etc/redpolitika/rules \
	PLUGINS_DIR=/etc/redpolitika/plugins \
	DB_DRIVER=sqlite \
	DB_DSN=file:/data/redpolitika.db?cache=shared&_journal_mode=WAL

ENTRYPOINT ["/entrypoint.sh"]
