# Simple Store — Go gRPC Microservice

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Buf](https://img.shields.io/badge/Buf-1.65-2F67B6?style=flat-square&logo=buf)](https://buf.build/)
[![gRPC](https://img.shields.io/badge/gRPC-1.82-244c5a?style=flat-square&logo=grpc)](https://grpc.io/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Buf CI](https://github.com/ilramdhan/simple-store-go-grpc/actions/workflows/buf.yml/badge.svg)](https://github.com/ilramdhan/simple-store-go-grpc/actions/workflows/buf.yml)

A production-ready **Go gRPC microservice monorepo** for a simple product catalog store.  
Designed with Clean Architecture, Protocol Buffers v3, and grpc-gateway for dual gRPC + REST exposure.

</div>

---

## 📑 Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Proto Design](#proto-design)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [API Reference](#api-reference)
- [CI/CD](#cicd)
- [Roadmap](#roadmap)

---

## Overview

`simple-store-go-grpc` is a monorepo demonstrating best practices for building Go microservices with gRPC. It exposes a **ProductService** that supports full CRUD operations and is accessible via both **native gRPC** and **REST/HTTP** (via grpc-gateway HTTP transcoding).

Key characteristics:
- **Schema-first**: All APIs are defined in `.proto` files — the single source of truth
- **Versioned APIs**: Proto packages use `v1` versioning for backward-compatible evolution
- **Dual transport**: One service, two protocols (gRPC + REST) via grpc-gateway
- **OpenAPI docs**: Swagger spec auto-generated from proto annotations
- **Monorepo**: Shared proto definitions, independent service Go modules

---

## Tech Stack

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| **Language** | [Go](https://go.dev/) | 1.26 | Service implementation |
| **RPC Framework** | [gRPC](https://grpc.io/) | 1.82 | High-performance RPC transport |
| **Schema** | [Protocol Buffers v3](https://protobuf.dev/) | — | API contract definition |
| **Proto Tooling** | [Buf CLI](https://buf.build/) | 1.65 | Linting, generation, breaking change detection |
| **HTTP Gateway** | [grpc-gateway](https://grpc-ecosystem.github.io/grpc-gateway/) | 2.29 | REST ↔ gRPC transcoding |
| **API Docs** | [OpenAPI v2 (Swagger)](https://swagger.io/) | — | Auto-generated from proto |
| **Database** | [PostgreSQL](https://www.postgresql.org/) | 16+ | Primary data store (planned) |
| **Migration** | [golang-migrate](https://github.com/golang-migrate/migrate) | 4.19 | Schema version management |
| **Validation** | [go-playground/validator](https://github.com/go-playground/validator) | 10.x | Request validation |
| **CI/CD** | [GitHub Actions](https://github.com/features/actions) | — | Automated lint & BSR push |

---

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────┐
│                    API Clients                      │
│          gRPC Client         REST / HTTP Client     │
└──────────────┬───────────────────────┬──────────────┘
               │ gRPC (port 50051)     │ HTTP (port 8080)
               │                       │
┌──────────────▼───────────────────────▼───────────────┐
│                  product-service                     │
│  ┌────────────────────────────────────────────────┐  │
│  │           gRPC Server (Handler Layer)          │  │
│  │    Validates requests · Maps proto ↔ domain    │  │
│  │              ┌─────────────────┐               │  │
│  │              │  grpc-gateway   │               │  │
│  │              │  (HTTP proxy)   │               │  │
│  │              └─────────────────┘               │  │
│  └────────────────────────┬───────────────────────┘  │
│                           │                          │
│  ┌────────────────────────▼───────────────────────┐  │
│  │                UseCase Layer                   │  │
│  │      Business rules · Orchestration            │  │
│  └────────────────────────┬───────────────────────┘  │
│                           │  (interface)             │
│  ┌────────────────────────▼───────────────────────┐  │
│  │              Repository Layer                  │  │
│  │         PostgreSQL (pgx driver)                │  │
│  └────────────────────────┬───────────────────────┘  │
│                           │                          │
└───────────────────────────┼──────────────────────────┘
                            │
                  ┌─────────▼─────────┐
                  │   PostgreSQL 16   │
                  └───────────────────┘
```

### Clean Architecture Layers

```
cmd/server/main.go          ← Entry point: wiring only, no business logic
│
├── internal/handler/grpc/  ← [Transport Layer]
│   · Validate & decode gRPC request
│   · Call UseCase
│   · Map domain → proto response
│
├── internal/usecase/       ← [Business Logic Layer]
│   · Business rules & orchestration
│   · Depends only on domain interfaces
│   · No knowledge of transport or DB
│
├── internal/domain/        ← [Domain Layer]
│   · Domain entity structs
│   · Repository interfaces (ports)
│   · Domain-specific errors
│
└── internal/repository/    ← [Data Access Layer]
    · Implements domain repository interfaces
    · PostgreSQL via pgx
    · SQL migrations in /migrations
```

**Dependency rule**: dependencies flow strictly **inward**.  
`handler` → `usecase` → `domain` ← `repository`  
No layer may import from a layer outside it.

### Proto Generation Pipeline

```
proto/**/*.proto
       │
       ▼  buf generate
       │
       ├─→ gen/go/**/**.pb.go          (protobuf message types)
       ├─→ gen/go/**/**_grpc.pb.go     (gRPC client/server stubs)
       ├─→ gen/go/**/**pb.gw.go        (grpc-gateway HTTP handlers)
       └─→ gen/openapiv2/api.swagger.json  (OpenAPI v2 spec)
```

---

## Project Structure

```
simple-store-go-grpc/
│
├── .github/
│   └── workflows/
│       └── buf.yml                 # CI: proto lint + BSR push
│
├── proto/                          # Schema-first API definitions (source of truth)
│   ├── common/
│   │   └── v1/
│   │       └── common.proto        # Shared: BaseResponse, Pagination, AuditInfo
│   └── product/
│       └── v1/
│           └── product.proto       # ProductService RPC + HTTP annotations
│
├── gen/                            # ⚠️ AUTO-GENERATED — do not edit manually
│   ├── go/
│   │   ├── common/v1/              # Generated common Go types
│   │   └── product/v1/             # Generated product Go types + gRPC stubs + gateway
│   └── openapiv2/
│       └── api.swagger.json        # Generated Swagger/OpenAPI docs
│
├── services/
│   └── product-service/            # Standalone Go module (independent deployable)
│       ├── cmd/
│       │   └── server/
│       │       └── main.go         # Entry point (DI wiring)
│       ├── internal/
│       │   ├── config/             # Config struct + env loader
│       │   ├── domain/             # Domain entities, interfaces, errors
│       │   ├── usecase/            # Business logic (pure Go, no I/O)
│       │   ├── handler/
│       │   │   └── grpc/           # gRPC request handlers
│       │   └── repository/
│       │       ├── postgres/       # PostgreSQL implementation
│       │       └── migrations/     # SQL migration files
│       ├── go.mod
│       └── go.sum
│
├── buf.yaml                        # Buf workspace config (v2)
├── buf.gen.yaml                    # Buf code generation config (v2)
├── buf.lock                        # Locked buf dependency versions
├── go.mod                          # Root module (workspace-level)
├── Makefile                        # Developer task automation
└── README.md                       # This file
```

---

## Proto Design

### Design Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| **Price type** | `int64 price_cents` | Floating-point is unsafe for money (`0.1 + 0.2 ≠ 0.3`) |
| **Timestamps** | `google.protobuf.Timestamp` | Type-safe, timezone-aware, RFC 3339 serialization |
| **Partial update** | `google.protobuf.FieldMask` | Properly distinguishes "not sent" from "sent as zero" |
| **HTTP paths** | Leading `/v1/products` | Required by grpc-gateway HTTP transcoding spec |
| **Pagination** | Page + PageSize | Simple, stateless, good enough for catalog use-cases |
| **Error envelope** | `BaseResponse` wrapper | Consistent error shape across all RPCs |

### Common Types (`common/v1/common.proto`)

```protobuf
message BaseResponse {
  int32 http_status_code      = 1;  // e.g. 200, 400, 404, 500
  bool  is_success            = 2;
  string message              = 3;
  repeated ValidationError validation_errors = 4;
}

message AuditInfo {
  google.protobuf.Timestamp created_at = 1;
  string created_by                    = 2;
  google.protobuf.Timestamp updated_at = 3;
  string updated_by                    = 4;
}
```

---

## Getting Started

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.26 | [go.dev/dl](https://go.dev/dl/) |
| Buf CLI | ≥ 1.65 | [buf.build/docs/installation](https://buf.build/docs/installation) |
| Docker & Compose | Latest | [docker.com](https://docker.com) |
| golangci-lint | Latest | [golangci-lint.run](https://golangci-lint.run/usage/install/) |

### Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/ilramdhan/simple-store-go-grpc.git
cd simple-store-go-grpc

# 2. Update buf dependencies
make proto-dep-update

# 3. Regenerate proto code (or use the committed gen/ files)
make proto-generate

# 4. Tidy Go dependencies
make tidy

# 5. (Coming soon) Start with Docker
# make docker-up
```

### Available Make Targets

```bash
make help              # Show all available targets

# Proto
make proto-lint        # Lint proto files with buf
make proto-generate    # Generate Go code + OpenAPI docs from proto
make proto-dep-update  # Update buf.lock (after changing buf.yaml deps)
make proto-breaking    # Check for breaking proto changes vs main branch
make proto-all         # Lint + generate

# Go
make build             # Build product-service binary
make test              # Run tests with race detector + coverage
make tidy              # go mod tidy for all modules
make lint              # Run golangci-lint

# Misc
make clean             # Remove all generated code (gen/ directory)
```

---

## API Reference

The full OpenAPI v2 (Swagger) spec is generated at [`gen/openapiv2/api.swagger.json`](gen/openapiv2/api.swagger.json).

### ProductService Endpoints

| Method | Path | RPC | Description |
|--------|------|-----|-------------|
| `POST` | `/v1/products` | `CreateProduct` | Create a new product |
| `GET` | `/v1/products` | `ListProducts` | List products (paginated) |
| `GET` | `/v1/products/{id}` | `GetProduct` | Get a product by ID |
| `PATCH` | `/v1/products/{id}` | `UpdateProduct` | Partial update via FieldMask |
| `DELETE` | `/v1/products/{id}` | `DeleteProduct` | Delete a product |

### Example: Create Product

**gRPC** (using `grpcurl`):
```bash
grpcurl -plaintext -d '{
  "name": "Wireless Headphones",
  "description": "Noise-cancelling over-ear headphones",
  "price_cents": 29999,
  "stock": 50
}' localhost:50051 product.v1.ProductService/CreateProduct
```

**REST** (via grpc-gateway):
```bash
curl -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wireless Headphones",
    "description": "Noise-cancelling over-ear headphones",
    "price_cents": 29999,
    "stock": 50
  }'
```

**Response**:
```json
{
  "base": {
    "http_status_code": 201,
    "is_success": true,
    "message": "Product created successfully"
  },
  "product": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Wireless Headphones",
    "description": "Noise-cancelling over-ear headphones",
    "price_cents": "29999",
    "stock": 50,
    "audit": {
      "created_at": "2026-07-08T03:00:00Z",
      "created_by": "system"
    }
  }
}
```

### Example: Partial Update (FieldMask)

```bash
# Only update name and stock — price_cents is unchanged
curl -X PATCH http://localhost:8080/v1/products/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "update_mask": "name,stock",
    "product": {
      "name": "Premium Wireless Headphones",
      "stock": 100
    }
  }'
```

---

## CI/CD

### GitHub Actions Workflows

| Workflow | Trigger | Steps |
|----------|---------|-------|
| `buf.yml` | Push to `main`, PRs | Proto lint → Breaking change check (PR only) → BSR push (main only) |

### Required Repository Secrets

| Secret | Description |
|--------|-------------|
| `BUF_TOKEN` | Buf Schema Registry token (for pushing proto to BSR) |

---

## Roadmap

### ✅ Done
- [x] Proto schema design with best practices (FieldMask, Timestamp, price_cents)
- [x] Buf v2 workspace configuration
- [x] gRPC-Gateway HTTP transcoding annotations
- [x] OpenAPI v2 Swagger spec generation
- [x] CI workflow for proto linting and BSR publishing

### 🚧 In Progress
- [ ] `cmd/server/main.go` — server wiring & graceful shutdown
- [ ] `internal/domain` — domain entities and repository interfaces
- [ ] `internal/usecase` — business logic implementation
- [ ] `internal/repository/postgres` — PostgreSQL data access layer
- [ ] `internal/handler/grpc` — gRPC request handlers
- [ ] SQL migrations for `products` table

### 📋 Planned
- [ ] Docker & docker-compose for local development (PostgreSQL)
- [ ] gRPC interceptors (logging, recovery, request ID)
- [ ] JWT authentication middleware
- [ ] `golangci-lint` configuration (`.golangci.yml`)
- [ ] Go CI workflow (build, test, lint)
- [ ] OpenTelemetry tracing
- [ ] Prometheus metrics endpoint
- [ ] Integration tests

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/your-feature`)
3. **Never edit files in `gen/`** — they are auto-generated
4. Run `make proto-lint` before committing proto changes
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
6. Open a Pull Request

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
