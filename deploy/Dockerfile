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

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /redpolitika /usr/local/bin/redpolitika
COPY --from=frontend /frontend/out /frontend/out
COPY backend/rules/ /etc/redpolitika/rules/
EXPOSE 8080
ENV STATIC_DIR=/frontend/out
ENV RULES_DIR=/etc/redpolitika/rules
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["redpolitika"]
