# Frontend builder
FROM node:24-alpine AS frontend
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Go builder
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /redpolitika ./cmd/api

# Runtime — CE image, no bundled rules
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata wget \
	&& mkdir -p /frontend/out /etc/redpolitika/rules /etc/redpolitika/plugins /data

COPY --from=builder /redpolitika /usr/local/bin/redpolitika
COPY --from=frontend /frontend/out /frontend/out

# Defaults for a runnable Hub image (override via -e / compose)
ENV PORT=8080 \
	LOG_LEVEL=info \
	ENVIRONMENT=production \
	STATIC_DIR=/frontend/out \
	RULES_DIR=/etc/redpolitika/rules \
	PLUGINS_DIR=/etc/redpolitika/plugins \
	DB_DRIVER=sqlite \
	DB_DSN=file:/data/redpolitika.db?cache=shared&_journal_mode=WAL

EXPOSE 8080
VOLUME ["/data", "/etc/redpolitika/rules", "/etc/redpolitika/plugins"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- "http://127.0.0.1:${PORT}/health" || exit 1

ENTRYPOINT ["redpolitika"]
