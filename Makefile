.PHONY: proto-lint proto-breaking proto-generate proto-all \
        build test lint tidy clean help

# ─── Variables ────────────────────────────────────────────────────────────────

PRODUCT_SVC_DIR := services/product-service
PROTO_GEN_DIR   := gen

# ─── Proto ────────────────────────────────────────────────────────────────────

## proto-lint: Run buf linter against all proto files
proto-lint:
	buf lint

## proto-breaking: Check for breaking changes against main branch
proto-breaking:
	buf breaking --against 'https://github.com/ilramdhan/simple-store-go-grpc.git#branch=main'

## proto-generate: Generate Go stubs, gRPC-Gateway, and OpenAPI docs from proto
proto-generate:
	buf generate

## proto-dep-update: Update buf dependencies and regenerate buf.lock
proto-dep-update:
	buf dep update

## proto-all: Run lint, then generate (use proto-breaking separately in CI)
proto-all: proto-lint proto-generate

# ─── Go ───────────────────────────────────────────────────────────────────────

## build: Build the product-service binary
build:
	cd $(PRODUCT_SVC_DIR) && go build -v ./cmd/server/...

## test: Run all tests with race detector and coverage
test:
	cd $(PRODUCT_SVC_DIR) && go test -race -count=1 -coverprofile=coverage.out ./...
	cd $(PRODUCT_SVC_DIR) && go tool cover -func=coverage.out

## tidy: Tidy go.mod and go.sum for all modules
tidy:
	go mod tidy
	cd $(PRODUCT_SVC_DIR) && go mod tidy

## lint: Run golangci-lint
lint:
	golangci-lint run ./$(PRODUCT_SVC_DIR)/...

# ─── Codegen & Clean ──────────────────────────────────────────────────────────

## clean: Remove all generated proto artifacts
clean:
	rm -rf $(PROTO_GEN_DIR)

# ─── Make Migrations ──────────────────────────────────────────────────────────

## migration: Generate skeleton file migrasi baru (make migration svc=user-service name=create_users)
migration:
	@if [ -z "$(svc)" ] || [ -z "$(name)" ]; then \
		echo "\033[31m[ERROR] Parameter 'svc' (service) dan 'name' wajib diisi!\033[0m"; \
		echo "Cara penggunaan : make migration svc=<nama-service> name=<nama-tabel>"; \
		echo "Contoh          : make migration svc=product-service name=create_products"; \
		exit 1; \
	fi
	@echo "\033[32mMembuat file migrasi '$(name)' untuk service '$(svc)'...\033[0m"
	migrate create -ext sql -dir services/$(svc)/internal/repository/postgres/migrations -format "20060102150405" $(name)

# ─── Help ─────────────────────────────────────────────────────────────────────

## help: Print this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'