# redpolitika CE — build helpers

MODULE   := github.com/drupaldoesnotexists/redpolitika/ce
INFO     := $(MODULE)/internal/buildinfo

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LICENSE    ?= BSL-1.1

LDFLAGS := -s -w \
	-X $(INFO).Version=$(VERSION) \
	-X $(INFO).Commit=$(COMMIT) \
	-X $(INFO).BuildTime=$(BUILD_TIME) \
	-X $(INFO).License=$(LICENSE)

.PHONY: build backend frontend docker version

version:
	@echo "version=$(VERSION) commit=$(COMMIT) build_time=$(BUILD_TIME)"

backend:
	cd backend && CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/redpolitika ./cmd/api

frontend:
	cd frontend && npm ci && npm run build

build: backend frontend

docker:
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg LICENSE="$(LICENSE)" \
		-f deploy/Dockerfile \
		-t redpolitika-ce:local \
		.
