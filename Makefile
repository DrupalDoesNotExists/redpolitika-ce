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

.PHONY: build backend frontend docker version proto-gen proto-gen-backend proto-gen-plugin

version:
	@echo "version=$(VERSION) commit=$(COMMIT) build_time=$(BUILD_TIME)"

# Generate protobuf Go code from .proto files
BACKEND_PROTO_DIRS := backend/proto/detect backend/proto/fix backend/proto/identity backend/proto/llm backend/proto/migrator backend/proto/pages
PLUGIN_PROTO_DIRS  := ce-plugins/pages/proto/identity ce-plugins/pages/proto/pages

proto-gen-backend:
	@for dir in $(BACKEND_PROTO_DIRS); do \
		proto="$$(ls $$dir/*.proto 2>/dev/null)"; \
		[ -n "$$proto" ] && cd $$dir && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative *.proto && cd - >/dev/null || true; \
	done

proto-gen-plugin:
	cd ce-plugins/pages && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/identity/identity.proto proto/pages/pages.proto

proto-gen: proto-gen-backend proto-gen-plugin

backend: proto-gen
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
