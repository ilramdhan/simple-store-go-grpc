# 📘 Go gRPC Microservice — Production Guide
### `simple-store-go-grpc` — Panduan Komprehensif Production-Grade

> **Untuk siapa**: Developer yang ingin belajar membangun Go microservice production-ready dengan gRPC, Clean Architecture, Domain-Driven Design (DDD), dan best practices industri terkini.
> **Repo**: `github.com/ilramdhan/simple-store-go-grpc`

---

## 🗺️ Peta Perjalanan (Table of Contents)

**BAGIAN I: FONDASI**
- Phase 1: Prasyarat & Tooling
- Phase 2: Arsitektur & Design Decisions
- Phase 3: Monorepo Setup
- Phase 4: Proto Definition (Schema-First)
- Phase 5: Code Generation

**BAGIAN II: IMPLEMENTASI SERVICE**
- Phase 6: Domain Layer (DDD Core)
- Phase 7: Config Layer
- Phase 8: Structured Logging
- Phase 9: Repository Layer
- Phase 10: Usecase Layer
- Phase 11: Handler Layer (gRPC)
- Phase 12: Server Wiring & DI

**BAGIAN III: PRODUCTION HARDENING**
- Phase 13: gRPC Interceptors
- Phase 14: Authentication & Authorization
- Phase 15: Health Checks & Readiness
- Phase 16: Observability
- Phase 17: Security Hardening
- Phase 18: Error Handling Strategy

**BAGIAN IV: TESTING**
- Phase 19: Unit Testing
- Phase 20: Integration Testing
- Phase 21: E2E & Manual Testing

**BAGIAN V: MULTI-SERVICE & DEPLOYMENT**
- Phase 22: Second Service (Order Service)
- Phase 23: Inter-Service Communication
- Phase 24: API Gateway
- Phase 25: Docker & Docker Compose
- Phase 26: CI/CD Pipeline
- Phase 27: Kubernetes Basics

**BAGIAN VI: REFERENSI (Appendix)**
- A. Kenapa Bukan GORM?
- B. Kenapa Bukan Viper?
- C. Kenapa Bukan Zap/Logrus?
- D. Makefile Lengkap
- E. golangci-lint Config
- F. Dependency Diagram
- G. Checklist Production Readiness
- H. Glossary

---

## Phase 1: Prasyarat & Tooling

Sebelum menulis kode, kita harus menyiapkan _tools_ yang menjadi standar di ekosistem Go modern.

### 1.1 Go 1.22+
Pastikan Anda menggunakan Go versi 1.22 atau lebih baru. Go 1.22 membawa peningkatan signifikan pada `net/http` router (walau kita pakai gRPC, ini berguna untuk gateway) dan perbaikan loop `for`.

```bash
go version
# Output: go version go1.22.x linux/amd64
```

### 1.2 Buf CLI
Buf adalah tool modern untuk manajemen Protocol Buffers. Menggantikan kerumitan instalasi `protoc` dan berbagai pluginnya secara manual. Buf memastikan _deterministic generation_ dan memiliki linter built-in.

```bash
# Install via Go
go install github.com/bufbuild/buf/cmd/buf@latest

buf --version
# Output: 1.x.x
```

### 1.3 golangci-lint
Linter standar industri untuk Go yang menggabungkan puluhan linter (seperti `errcheck`, `gosec`, `staticcheck`) menjadi satu perintah cepat.

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest

golangci-lint --version
```

### 1.4 grpcurl
Seperti `curl` tapi untuk gRPC. Sangat penting untuk testing manual gRPC API karena gRPC menggunakan HTTP/2 dan format binary (Protobuf), sehingga tidak bisa di-test dengan curl biasa.

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### 1.5 Docker & Docker Compose
Dibutuhkan untuk menjalankan dependensi seperti PostgreSQL, Jaeger (untuk tracing), dan nantinya menjalankan service kita di dalam container.

---

## Phase 2: Arsitektur & Design Decisions

### 2.1 Clean Architecture & DDD
Proyek ini menggunakan **Clean Architecture** yang dipadukan dengan prinsip **Domain-Driven Design (DDD)**.

```text
(Lapisan Luar / Transport & DB)   ──► Handler (gRPC), Repo (PostgreSQL)
        ▼
(Lapisan Aplikasi / Business)     ──► Usecase
        ▼
(Lapisan Inti / Core)             ──► Domain (Entities, Repository Interface)
```

**Aturan Emas (Dependency Rule):**
> Lapisan luar bergantung pada lapisan dalam. Lapisan dalam **TIDAK PERNAH** tahu tentang lapisan luar.
> - Domain tidak tahu tentang Usecase atau Repo.
> - Usecase tidak tahu tentang Handler (HTTP/gRPC) atau PostgreSQL.

**Konsep DDD yang dipakai:**
- **Entity**: Objek bisnis utama yang memiliki identitas (`Product`, `Order`).
- **Value Object**: Objek tanpa identitas unik, dinilai dari atributnya.
- **Repository Interface**: Kontrak untuk menyimpan dan mengambil _Aggregate_.

### 2.2 Technology Choices (Kenapa pilih X bukan Y?)

- **Kenapa `pgx` dan bukan `GORM`?**
  GORM memaksa kita menaruh tag database di struct Domain, melanggar Clean Architecture. `pgx` sangat cepat, dan kita memegang kendali penuh atas SQL (lihat _Appendix A_).
- **Kenapa `koanf` dan bukan `Viper`?**
  Viper sering digunakan sebagai global state (`viper.GetString("...")`), yang rentan bug dan sulit di-test. `koanf` lebih ringan, modern, dan mendorong injeksi konfigurasi lewat struct (lihat _Appendix B_).
- **Kenapa `slog` dan bukan `Zap/Logrus`?**
  Sejak Go 1.21, `log/slog` menjadi bagian dari standard library. Performanya sangat baik dan memiliki format JSON bawaan. Tidak perlu external dependency (lihat _Appendix C_).

### 2.3 Layout Proyek
Struktur yang kita gunakan mengadopsi _Standard Go Project Layout_ namun dioptimasi untuk Monorepo yang berisi banyak microservice. Setiap service memiliki modul Go sendiri (`go.mod`) di dalam folder `services/`.

---

## Phase 3: Monorepo Setup

Mari kita buat struktur direktorinya.

```bash
mkdir simple-store-go-grpc
cd simple-store-go-grpc

# Buat folder utama
mkdir -p proto/common/v1 proto/product/v1 proto/order/v1
mkdir -p services/product-service/cmd/server
mkdir -p services/order-service/cmd/server
mkdir -p services/api-gateway/cmd/server
```

### 3.1 Go Workspace (`go.work`)
Karena kita menggunakan monorepo dengan beberapa modul Go (satu root untuk proto, dan modul terpisah untuk tiap service), kita gunakan `go.work` agar IDE (seperti GoLand atau VSCode) bisa mengenali semuanya.

**File**: `go.work`
```go
go 1.22.0

use (
	.
	./services/api-gateway
	./services/order-service
	./services/product-service
)
```

### 3.2 Root Module & Service Module

**File**: `go.mod` (di root proyek - hanya untuk kode auto-generated Proto)
```go
module github.com/ilramdhan/simple-store-go-grpc

go 1.22.0
```

**File**: `services/product-service/go.mod`
```go
module github.com/ilramdhan/simple-store-go-grpc/services/product-service

go 1.22.0
```
*(Buat file serupa untuk order-service dan api-gateway).*

### 3.3 Gitignore

**File**: `.gitignore`
```text
# Binaries
/bin/
*.exe
*.exe~
*.dll
*.so
*.dylib
server

# Output
/gen/
coverage.out

# IDE
.idea/
.vscode/
*.swp
*.swo

# Environment
.env

# Go workspace
go.work
go.work.sum
```

---

## Phase 4: Proto Definition (Schema-First)

**Schema-First** berarti kita mendefinisikan API sebagai "kontrak" dalam file `.proto` sebelum menulis kode Go apapun. Ini memisahkan desain API dari implementasi.

### 4.1 Common Proto
Berisi message yang dipakai ulang oleh berbagai service.

**File**: `proto/common/v1/common.proto`
```protobuf
syntax = "proto3";

package common.v1;

import "google/protobuf/timestamp.proto";

// BaseResponse adalah pembungkus standar untuk response.
message BaseResponse {
  int32 http_status_code = 1;
  bool  is_success       = 2;
  string message         = 3;
  repeated ValidationError validation_errors = 4;
}

message ValidationError {
  string field   = 1;
  string message = 2;
}

message PaginationRequest {
  int32 page      = 1;
  int32 page_size = 2;
}

message PaginationResponse {
  int32 current_page = 1;
  int32 page_size    = 2;
  int64 total_items  = 3;
  int32 total_pages  = 4;
}

// AuditInfo standar untuk metadata.
message AuditInfo {
  google.protobuf.Timestamp created_at = 1;
  string                    created_by = 2;
  google.protobuf.Timestamp updated_at = 3;
  string                    updated_by = 4;
}
```

### 4.2 Product Proto

**File**: `proto/product/v1/product.proto`
```protobuf
syntax = "proto3";

package product.v1;

import "common/v1/common.proto";
import "google/api/annotations.proto";
import "google/protobuf/field_mask.proto";

message Product {
  string id          = 1;
  string name        = 2;
  string description = 3;
  int64  price_cents = 4;
  int32  stock       = 5;
  common.v1.AuditInfo audit = 6;
}

message CreateProductRequest {
  string name        = 1;
  string description = 2;
  int64  price_cents = 3;
  int32  stock       = 4;
}

message CreateProductResponse {
  common.v1.BaseResponse base    = 1;
  Product                product = 2;
}

message GetProductRequest {
  string id = 1;
}

message GetProductResponse {
  common.v1.BaseResponse base    = 1;
  Product                product = 2;
}

// Menggunakan FieldMask untuk partial update (PATCH)
message UpdateProductRequest {
  string                    id          = 1;
  google.protobuf.FieldMask update_mask = 2;
  Product                   product     = 3;
}

message UpdateProductResponse {
  common.v1.BaseResponse base    = 1;
  Product                product = 2;
}

message DeleteProductRequest {
  string id = 1;
}

message DeleteProductResponse {
  common.v1.BaseResponse base = 1;
}

message ListProductsRequest {
  common.v1.PaginationRequest pagination = 1;
}

message ListProductsResponse {
  common.v1.BaseResponse       base       = 1;
  repeated Product             products   = 2;
  common.v1.PaginationResponse pagination = 3;
}

service ProductService {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse) {
    option (google.api.http) = {
      post: "/v1/products"
      body: "*"
    };
  }

  rpc GetProduct(GetProductRequest) returns (GetProductResponse) {
    option (google.api.http) = { get: "/v1/products/{id}" };
  }

  rpc UpdateProduct(UpdateProductRequest) returns (UpdateProductResponse) {
    option (google.api.http) = {
      patch: "/v1/products/{id}"
      body: "*"
    };
  }

  rpc DeleteProduct(DeleteProductRequest) returns (DeleteProductResponse) {
    option (google.api.http) = { delete: "/v1/products/{id}" };
  }

  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {
    option (google.api.http) = { get: "/v1/products" };
  }
}
```

### 4.3 Order Proto (Untuk Multi-Service)

**File**: `proto/order/v1/order.proto`
```protobuf
syntax = "proto3";

package order.v1;

import "common/v1/common.proto";
import "google/api/annotations.proto";

enum OrderStatus {
  ORDER_STATUS_UNSPECIFIED = 0;
  ORDER_STATUS_PENDING     = 1;
  ORDER_STATUS_CONFIRMED   = 2;
  ORDER_STATUS_SHIPPED     = 3;
  ORDER_STATUS_DELIVERED   = 4;
  ORDER_STATUS_CANCELLED   = 5;
}

message Order {
  string id          = 1;
  string product_id  = 2;
  int32  quantity    = 3;
  int64  total_cents = 4;
  OrderStatus status = 5;
  common.v1.AuditInfo audit = 6;
}

message CreateOrderRequest {
  string product_id = 1;
  int32  quantity   = 2;
}

message CreateOrderResponse {
  common.v1.BaseResponse base  = 1;
  Order                  order = 2;
}

// (Get, List, UpdateStatus requests dihilangkan untuk brevity di bab ini, tapi asumsikan ada)

service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse) {
    option (google.api.http) = {
      post: "/v1/orders"
      body: "*"
    };
  }
}
```

---

## Phase 5: Code Generation

Kita menggunakan Buf untuk mengubah file `.proto` menjadi kode Go.

### 5.1 Konfigurasi Buf

**File**: `buf.yaml` (Konfigurasi Workspace)
```yaml
version: v2
modules:
  - path: proto
    name: buf.build/ilramdhan/simple-store-go-grpc
deps:
  - buf.build/googleapis/googleapis
  - buf.build/protocolbuffers/wellknowntypes
lint:
  use: [STANDARD]
  except: [PACKAGE_VERSION_SUFFIX]
breaking:
  use: [FILE]
```

**File**: `buf.gen.yaml` (Konfigurasi Code Generation)
```yaml
version: v2
managed:
  enabled: true
  override:
    # Managed mode menyuntikkan go_package otomatis ke semua file proto
    - file_option: go_package_prefix
      value: github.com/ilramdhan/simple-store-go-grpc/gen/go
plugins:
  # 1. Struct Go (protoc-gen-go)
  - remote: buf.build/protocolbuffers/go:v1.36.11
    out: gen/go
    opt: [paths=source_relative]

  # 2. Interface Server/Client gRPC (protoc-gen-go-grpc)
  - remote: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
      - require_unimplemented_servers=true

  # 3. HTTP Gateway (protoc-gen-grpc-gateway)
  - remote: buf.build/grpc-ecosystem/gateway:v2.29.0
    out: gen/go
    opt:
      - paths=source_relative
      - generate_unbound_methods=true

  # 4. OpenAPI / Swagger documentation
  - remote: buf.build/grpc-ecosystem/openapiv2:v2.29.0
    out: gen/openapiv2
    opt:
      - generate_unbound_methods=true
      - allow_merge=true
      - merge_file_name=api
inputs:
  - directory: proto
```

### 5.2 Generate Kode

Jalankan perintah ini di root direktori:

```bash
# Update dependensi buf (seperti googleapis)
buf dep update

# Pastikan tidak ada error lint (misal snake_case untuk field)
buf lint

# Generate kode!
buf generate
```

Setelah `buf generate`, folder `gen/go` akan berisi struct, interface gRPC server, dan HTTP multiplexer untuk API Gateway. File yang krusial bagi kita adalah interface `ProductServiceServer` di `gen/go/product/v1/product_grpc.pb.go`. Ini adalah kontrak yang HARUS diimplementasikan oleh Handler kita nantinya.

> ⚠️ **PENTING:** Jangan pernah mengedit isi folder `gen/` secara manual karena akan tertimpa saat Anda menjalankan `buf generate` lagi.


# BAGIAN II: IMPLEMENTASI SERVICE

Bagian ini adalah inti dari aplikasi kita. Kita akan membangun `product-service` lapis demi lapis dari dalam ke luar mengikuti prinsip Clean Architecture.

---

## Phase 6: Domain Layer (DDD Core)

**Domain Layer** adalah lapisan paling dalam. Di sini kita mendefinisikan objek bisnis (Entity), kontrak penyimpanan (Repository Interface), dan kontrak bisnis (Usecase Interface).

> ⚠️ **ATURAN MUTLAK:** Layer Domain HANYA BOLEH mengimpor standard library Go. Tidak boleh mengimpor package proto, driver database, atau framework HTTP.

### 6.1 Entity & Interfaces

**File**: `services/product-service/internal/domain/product.go`
```go
package domain

import (
	"context"
	"time"
)

// Product adalah entitas bisnis inti.
// Perhatikan bahwa kita menggunakan tipe data bawaan Go, bukan tipe dari protobuf.
type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64 // Harga dalam satuan terkecil (sen/rupiah) untuk presisi
	Stock       int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   string
	UpdatedBy   string
}

// ProductRepository adalah kontrak untuk layer persistence.
type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int32) ([]*Product, int64, error)
}

// ProductUsecase adalah kontrak untuk layer business logic.
// Handler nantinya akan memanggil interface ini.
type ProductUsecase interface {
	Create(ctx context.Context, input CreateProductInput) (*Product, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, input UpdateProductInput) (*Product, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int32) ([]*Product, int64, error)
}

// Input structs: digunakan sebagai argumen untuk Usecase.
// Struct ini polos (bukan protobuf) dan bisa diberikan tag validasi.
type CreateProductInput struct {
	Name        string `validate:"required,min=3,max=100"`
	Description string `validate:"max=500"`
	PriceCents  int64  `validate:"min=0"`
	Stock       int32  `validate:"min=0"`
}

type UpdateProductInput struct {
	Name        *string `validate:"omitempty,min=3,max=100"`
	Description *string `validate:"omitempty,max=500"`
	PriceCents  *int64  `validate:"omitempty,min=0"`
	Stock       *int32  `validate:"omitempty,min=0"`
}
```

### 6.2 Domain Errors (Sentinel Errors)

**File**: `services/product-service/internal/domain/errors.go`
```go
package domain

import "errors"

// Sentinel errors ini mendefinisikan masalah spesifik domain bisnis kita.
// Layer luar akan mengecek error ini dengan errors.Is() untuk menentukan HTTP/gRPC status code.

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrProductNameEmpty     = errors.New("product name cannot be empty")
	ErrProductPriceNegative = errors.New("product price cannot be negative")
	ErrProductStockNegative = errors.New("product stock cannot be negative")
)
```

---

## Phase 7: Config Layer

Kita menggunakan pola **12-Factor App**, di mana konfigurasi dibaca dari Environment Variables. Di V3 ini, kita menggunakan `koanf` alih-alih `Viper` (terlalu berat) atau `caarlos0/env` (terlalu sederhana).

### 7.1 Config Struct & Parser

**File**: `services/product-service/internal/config/config.go`
```go
package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

type Config struct {
	App      AppConfig      `koanf:"app"`
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	Log      LogConfig      `koanf:"log"`
}

type AppConfig struct {
	Env  string `koanf:"env"`
	Name string `koanf:"name"`
}

type ServerConfig struct {
	GRPCPort        int `koanf:"grpc_port"`
	HTTPGatewayPort int `koanf:"http_port"`
}

type DatabaseConfig struct {
	URL             string `koanf:"url"` // Dibaca dari env: DATABASE_URL
	MaxConns        int32  `koanf:"max_conns"`
	MinConns        int32  `koanf:"min_conns"`
}

type LogConfig struct {
	Level string `koanf:"level"`
}

// Load membaca default.yaml, lalu menimpanya dengan Environment Variables.
func Load(configPath string) (*Config, error) {
	// 1. Baca dari file YAML
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		// Toleransi jika file tak ada saat di produksi, asalkan ENV tersedia
		fmt.Printf("Warning: error loading config file: %v\n", err)
	}

	// 2. Baca dari ENV. Ganti "_" menjadi "." untuk map ke hierarki.
	// Contoh: DATABASE_URL -> database.url
	k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
```

### 7.2 File Konfigurasi Default

**File**: `services/product-service/internal/config/default.yaml`
```yaml
app:
  env: "development"
  name: "product-service"
server:
  grpc_port: 50051
  http_port: 8080
database:
  url: "" # Wajib di-inject via env var DATABASE_URL
  max_conns: 20
  min_conns: 5
log:
  level: "info"
```

---

## Phase 8: Structured Logging

Di produksi, log harus terstruktur (JSON) agar mudah ditelusuri oleh sistem seperti ELK, Loki, atau Datadog. Kita gunakan `log/slog` bawaan Go 1.21+.

**File**: `services/product-service/internal/logger/logger.go`
```go
package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string
const requestIDKey ctxKey = "request_id"

// InitLogger mengatur default logger global.
func InitLogger(levelStr, serviceName string) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Gunakan JSON handler
	handler := slog.NewJSONHandler(os.Stdout, opts).WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
	})

	slog.SetDefault(slog.New(handler))
}

// FromContext mengambil logger yang sudah disisipi request_id.
// Sangat berguna agar setiap baris log memiliki identitas request yang sama.
func FromContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		logger = logger.With(slog.String("request_id", reqID))
	}
	return logger
}
```

---

## Phase 9: Repository Layer

Layer ini bertugas mengimplementasikan `ProductRepository` menggunakan PostgreSQL via library `pgx`.

### 9.1 Migrations

**File**: `services/product-service/internal/repository/postgres/migrations/000001_create_products.up.sql`
```sql
CREATE TABLE products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    stock       INTEGER NOT NULL CHECK (stock >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(100) NOT NULL,
    updated_by  VARCHAR(100) NOT NULL,
    deleted_at  TIMESTAMPTZ -- NULL means active
);

-- Indexing untuk query yang sering dilakukan
CREATE INDEX idx_products_deleted_at ON products(deleted_at);
```

### 9.2 Implementasi pgx

**File**: `services/product-service/internal/repository/postgres/product_repo.go`
```go
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

type productRepo struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) domain.ProductRepository {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *domain.Product) error {
	query := `
		INSERT INTO products (id, name, description, price_cents, stock, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	// Selalu gunakan parameterized query ($1, $2) untuk mencegah SQL Injection
	err := r.db.QueryRow(ctx, query,
		p.ID, p.Name, p.Description, p.PriceCents, p.Stock, p.CreatedBy, p.UpdatedBy,
	).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("repo.Create: %w", err)
	}
	return nil
}

func (r *productRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `
		SELECT id, name, description, price_cents, stock, created_at, updated_at, created_by, updated_by
		FROM products
		WHERE id = $1 AND deleted_at IS NULL
	`
	p := &domain.Product{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.PriceCents, &p.Stock,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Translate db error menjadi domain error
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("repo.GetByID: %w", err)
	}
	return p, nil
}

func (r *productRepo) Update(ctx context.Context, p *domain.Product) error {
	query := `
		UPDATE products
		SET name = $1, description = $2, price_cents = $3, stock = $4, updated_by = $5, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		p.Name, p.Description, p.PriceCents, p.Stock, p.UpdatedBy, p.ID,
	).Scan(&p.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrProductNotFound
		}
		return fmt.Errorf("repo.Update: %w", err)
	}
	return nil
}

func (r *productRepo) Delete(ctx context.Context, id string) error {
	query := `UPDATE products SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repo.Delete: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *productRepo) List(ctx context.Context, page, pageSize int32) ([]*domain.Product, int64, error) {
	offset := (page - 1) * pageSize

	// ⚠️ MENGHINDARI RACE CONDITION (Solusi V3)
	// Kita gunakan Window Function `COUNT(*) OVER()` untuk mendapatkan total item
	// dan paginasi dalam SATU query, menghindari ketidaksinkronan data.
	query := `
		SELECT
			id, name, description, price_cents, stock, created_at, updated_at, created_by, updated_by,
			COUNT(*) OVER() AS total_items
		FROM products
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repo.List: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	var total int64

	for rows.Next() {
		p := &domain.Product{}
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.PriceCents, &p.Stock,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
			&total, // field ekstra dari window function
		)
		if err != nil {
			return nil, 0, fmt.Errorf("repo.List scan: %w", err)
		}
		products = append(products, p)
	}

	return products, total, nil
}
```

---

## Phase 10: Usecase Layer

Usecase memuat aturan bisnis. **V3 Kritis:** Usecase JANGAN PERNAH mengimpor package `productv1` (Proto). Usecase hanya menerima dan mengembalikan `domain.Product`.

**File**: `services/product-service/internal/usecase/product_usecase.go`
```go
package usecase

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
)

type productUsecase struct {
	repo     domain.ProductRepository
	validate *validator.Validate
}

func NewProductUsecase(repo domain.ProductRepository) domain.ProductUsecase {
	return &productUsecase{
		repo:     repo,
		validate: validator.New(),
	}
}

func (u *productUsecase) Create(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error) {
	// 1. Validasi Input
	if err := u.validate.Struct(input); err != nil {
		// Validasi gagal, return raw error (nanti di-format oleh handler)
		return nil, err
	}

	// 2. Terapkan logika bisnis / bentuk entitas
	p := &domain.Product{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		PriceCents:  input.PriceCents,
		Stock:       input.Stock,
		CreatedBy:   "user", // Dalam sistem asli, ambil dari ctx via Auth interceptor
		UpdatedBy:   "user",
	}

	// 3. Simpan via Repo
	if err := u.repo.Create(ctx, p); err != nil {
		logger.FromContext(ctx).Error("failed to create product", "error", err)
		return nil, err // Bisa wrap error jika perlu
	}

	return p, nil
}

func (u *productUsecase) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *productUsecase) Update(ctx context.Context, id string, input domain.UpdateProductInput) (*domain.Product, error) {
	if err := u.validate.Struct(input); err != nil {
		return nil, err
	}

	// Fetch current state
	p, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Terapkan partial update
	if input.Name != nil {
		p.Name = *input.Name
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.PriceCents != nil {
		p.PriceCents = *input.PriceCents
	}
	if input.Stock != nil {
		p.Stock = *input.Stock
	}
	p.UpdatedBy = "user" // Ideally from ctx

	if err := u.repo.Update(ctx, p); err != nil {
		logger.FromContext(ctx).Error("failed to update product", "error", err)
		return nil, err
	}

	return p, nil
}

func (u *productUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}

func (u *productUsecase) List(ctx context.Context, page, pageSize int32) ([]*domain.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return u.repo.List(ctx, page, pageSize)
}
```

---

## Phase 11: Handler Layer (gRPC)

Handler gRPC (atau HTTP) berada di ujung terluar. Tugas utamanya adalah menerjemahkan dunia luar (Proto `CreateProductRequest`) menjadi bahasa internal (Domain `CreateProductInput`), memanggil Usecase, lalu menerjemahkan balik Domain `Product` ke Proto `CreateProductResponse`.

### 11.1 Mapper

**File**: `services/product-service/internal/handler/grpc/mapper.go`
```go
package grpc

import (
	"errors"
	"math"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/common/v1"
	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

func domainToProtoProduct(p *domain.Product) *productv1.Product {
	if p == nil {
		return nil
	}
	return &productv1.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Stock:       p.Stock,
		Audit: &commonv1.AuditInfo{
			CreatedAt: timestamppb.New(p.CreatedAt),
			UpdatedAt: timestamppb.New(p.UpdatedAt),
			CreatedBy: p.CreatedBy,
			UpdatedBy: p.UpdatedBy,
		},
	}
}

// domainErrorToGRPC menerjemahkan error internal ke standar status gRPC
func domainErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	// 1. Tangani error Validasi
	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		// Kita konversi ke error InvalidArgument
		st := status.New(codes.InvalidArgument, "validation failed")
		// (Di sistem asli yang kompleks, Anda bisa attach struct detail ValidationError ke status gRPC)
		return st.Err()
	}

	// 2. Tangani error Sentinel Domain
	if errors.Is(err, domain.ErrProductNotFound) {
		return status.Errorf(codes.NotFound, "%v", err)
	}

	// 3. Fallback (Jangan pernah bocorkan error internal DB ke luar!)
	return status.Errorf(codes.Internal, "an internal error occurred")
}
```

### 11.2 Handler

**File**: `services/product-service/internal/handler/grpc/handler.go`
```go
package grpc

import (
	"context"

	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	commonv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/common/v1"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

var _ productv1.ProductServiceServer = (*Handler)(nil)

type Handler struct {
	productv1.UnimplementedProductServiceServer
	uc domain.ProductUsecase
}

func NewHandler(uc domain.ProductUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	input := domain.CreateProductInput{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		PriceCents:  req.GetPriceCents(),
		Stock:       req.GetStock(),
	}

	product, err := h.uc.Create(ctx, input)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &productv1.CreateProductResponse{
		Base: &commonv1.BaseResponse{
			HttpStatusCode: 201,
			IsSuccess:      true,
			Message:        "Product created successfully",
		},
		Product: domainToProtoProduct(product),
	}, nil
}

func (h *Handler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	product, err := h.uc.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &productv1.GetProductResponse{
		Base: &commonv1.BaseResponse{HttpStatusCode: 200, IsSuccess: true},
		Product: domainToProtoProduct(product),
	}, nil
}

// Implementasi List, Update, dan Delete mengikuti pola serupa:
// Extract req -> Call uc -> Handle error with domainErrorToGRPC -> Convert result
```

---

## Phase 12: Server Wiring & DI

Dependency Injection (DI) mengikat semua komponen (Config -> DB -> Repo -> Usecase -> Handler). Kita pisahkan logika server (lifecycle) dari `main.go`.

### 12.1 Server Lifecycle

**File**: `services/product-service/internal/server/server.go`
```go
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/config"
	grpchandler "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/handler/grpc"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/repository/postgres"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/usecase"
)

type App struct {
	cfg        *config.Config
	dbPool     *pgxpool.Pool
	grpcServer *grpc.Server
	httpServer *http.Server
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	// 1. Setup DB
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = cfg.Database.MinConns

	dbPool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	// 2. Setup Layers
	repo := postgres.NewProductRepository(dbPool)
	uc := usecase.NewProductUsecase(repo)
	handler := grpchandler.NewHandler(uc)

	// 3. Setup gRPC Server (Interceptors akan ditambahkan di Part 3)
	grpcServer := grpc.NewServer()
	productv1.RegisterProductServiceServer(grpcServer, handler)

	// 4. Setup gRPC-Gateway
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	_ = productv1.RegisterProductServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort), opts,
	)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPGatewayPort),
		Handler: mux,
	}

	return &App{
		cfg:        cfg,
		dbPool:     dbPool,
		grpcServer: grpcServer,
		httpServer: httpServer,
	}, nil
}

func (a *App) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.cfg.Server.GRPCPort))
	if err != nil {
		return err
	}

	// Run gRPC server asynchronously
	go func() {
		logger.FromContext(context.Background()).Info("gRPC server starting", "port", a.cfg.Server.GRPCPort)
		if err := a.grpcServer.Serve(lis); err != nil {
			logger.FromContext(context.Background()).Error("gRPC server failed", "error", err)
		}
	}()

	// Run HTTP gateway asynchronously
	go func() {
		logger.FromContext(context.Background()).Info("HTTP Gateway starting", "port", a.cfg.Server.HTTPGatewayPort)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.FromContext(context.Background()).Error("HTTP gateway failed", "error", err)
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) {
	logger.FromContext(ctx).Info("Shutting down gracefully...")
	_ = a.httpServer.Shutdown(ctx)
	a.grpcServer.GracefulStop()
	a.dbPool.Close()
	logger.FromContext(ctx).Info("Shutdown complete.")
}
```

### 12.2 Main Entrypoint

`main.go` sekarang sangat tipis dan bersih.

**File**: `services/product-service/cmd/server/main.go`
```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/config"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/server"
)

func main() {
	cfg, err := config.Load("internal/config/default.yaml")
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	logger.InitLogger(cfg.Log.Level, cfg.App.Name)
	ctx := context.Background()

	app, err := server.NewApp(ctx, cfg)
	if err != nil {
		logger.FromContext(ctx).Error("Failed to initialize app", "error", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		logger.FromContext(ctx).Error("Failed to run app", "error", err)
		os.Exit(1)
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	app.Shutdown(shutdownCtx)
}
```


---
# BAGIAN III: PRODUCTION HARDENING

Kode yang berjalan lancar di lokal belum tentu siap untuk _production_. Di bagian ini, kita akan menambahkan lapisan observabilitas, keamanan, dan ketahanan menggunakan gRPC Interceptor.

## Phase 13: gRPC Interceptors

Interceptor di gRPC mirip dengan _Middleware_ di HTTP. Mereka membungkus setiap pemanggilan fungsi (RPC).
Urutan eksekusi sangat penting. Urutan yang direkomendasikan:
`Recovery -> RequestID -> Logging -> Metrics -> Auth`

Letakkan interceptor di `services/product-service/internal/interceptor/`.

### 13.1 Recovery Interceptor
Mencegah aplikasi _crash_ jika terjadi panic di layer mana pun.

**File**: `recovery.go`
```go
package interceptor

import (
	"context"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
)

func UnaryRecovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if p := recover(); p != nil {
				logger.FromContext(ctx).Error("panic recovered", "panic", p, "stack", string(debug.Stack()))
				err = status.Errorf(codes.Internal, "Internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
```

### 13.2 Request ID Interceptor
Menjamin setiap _request_ memiliki ID unik untuk penelusuran (tracing) di log.

**File**: `request_id.go`
```go
package interceptor

import (
	"context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey string
const RequestIDKey ctxKey = "request_id"

func UnaryRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-request-id"); len(vals) > 0 {
				requestID = vals[0]
			}
		}
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		
		// Pasang juga di log context (berkaitan dengan pkg logger yang kita buat)
		return handler(ctx, req)
	}
}
```

### 13.3 Logging Interceptor
Mencatat metode yang dipanggil, durasi, dan status code.

**File**: `logging.go`
```go
package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
)

func UnaryLogging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		resp, err := handler(ctx, req)
		
		duration := time.Since(start)
		st, _ := status.FromError(err)
		
		logEntry := logger.FromContext(ctx).With(
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"status_code", st.Code().String(),
		)

		if err != nil {
			logEntry.Error("gRPC request failed", "error", err)
		} else {
			logEntry.Info("gRPC request success")
		}

		return resp, err
	}
}
```

Cara memasangnya di `server.go`:
```go
// import interceptor buatan sendiri
import "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/interceptor"
// ...
grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        interceptor.UnaryRecovery(),
        interceptor.UnaryRequestID(),
        interceptor.UnaryLogging(),
    ),
)
```

---

## Phase 14: Authentication & Authorization

Kita menggunakan JSON Web Token (JWT). Token dikirim oleh client melalui metadata `authorization: Bearer <token>`.

### 14.1 Auth Interceptor

**File**: `auth.go`
```go
package interceptor

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type ctxKey string
const UserClaimsKey ctxKey = "user_claims"

func UnaryAuth(secret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for health checks
		if strings.Contains(info.FullMethod, "Health/Check") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
		}

		tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
		
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Masukkan claims ke dalam context agar bisa dibaca Usecase
		ctx = context.WithValue(ctx, UserClaimsKey, claims)
		return handler(ctx, req)
	}
}
```

### 14.2 Authorization (RBAC) di Handler
Di Handler, Anda bisa mengambil context ini:
```go
claims, ok := ctx.Value(interceptor.UserClaimsKey).(*interceptor.Claims)
if !ok || claims.Role != "admin" {
    return nil, status.Error(codes.PermissionDenied, "admin access required")
}
```

---

## Phase 15: Health Checks & Readiness

Di lingkungan Kubernetes, _liveness_ dan _readiness_ probe sangat penting. gRPC memiliki standar health check protokol tersendiri (`grpc.health.v1`).

Implementasinya di `server.go`:
```go
import (
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Di dalam NewApp():
healthServer := health.NewServer()
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
healthpb.RegisterHealthServer(grpcServer, healthServer)
```

Untuk readiness (cek koneksi DB), jalankan goroutine yang melakukan `dbPool.Ping(ctx)` berkala dan update status health server menjadi `NOT_SERVING` jika DB mati.

---

## Phase 16: Observability (Metrics & Tracing)

Standar industri modern untuk observabilitas adalah **OpenTelemetry (OTel)**. OpenTelemetry menggabungkan Tracing, Metrics, dan Logging ke dalam satu standar (menggantikan cara lama yang memisahkan Prometheus dan Jaeger secara manual).

### 16.1 Tracing & Metrics dengan OpenTelemetry
Kita menggunakan _instrumentation library_ resmi untuk gRPC: `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`.

```go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

// Di server.go
grpcServer := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
    // ... interceptor lain
)
```
Setiap kali terjadi _request_, `otelgrpc` secara otomatis akan men-generate _Trace ID_, _Span ID_, dan metrik terkait performa gRPC (RED metrics: Rate, Errors, Duration), yang kemudian akan diekspor ke kolektor seperti OpenTelemetry Collector, Datadog, atau Grafana Tempo/Mimir.

---

## Phase 17: Security Hardening

Keamanan tidak boleh ditambahkan di akhir (afterthought). Di _production_, kita harus menerapkan prinsip *Zero-Trust*:

### 17.1 Transport Layer Security (mTLS / TLS)
Jangan pernah menjalankan gRPC tanpa enkripsi di _production public network_. Walau di _internal network_ (Kubernetes), mTLS (Mutual TLS) sangat disarankan.

```go
import "google.golang.org/grpc/credentials"

// Load sertifikat TLS
creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
if err != nil {
    log.Fatalf("Failed to setup TLS: %v", err)
}

grpcServer := grpc.NewServer(
    grpc.Creds(creds),
    // interceptors...
)
```

### 17.2 Rate Limiting
Mencegah serangan DDoS atau _brute-force_ pada API kita. Anda dapat menambahkan Interceptor Rate Limiter menggunakan algoritma _Token Bucket_ (misalnya library `golang.org/x/time/rate`).
Rate limiting biasanya juga dipasang di level infrastruktur via Ingress NGINX atau API Gateway (seperti Kong/Tyk).

### 17.3 Input Validation (Protovalidate)
Untuk menjamin payload gRPC aman sebelum masuk ke layer Usecase, standar terbaru adalah menggunakan `bufbuild/protovalidate`. Kita cukup menaruh constraint di file `.proto` dan _validator interceptor_ akan memblokir request yang kotor secara otomatis.

---

## Phase 18: Error Handling Strategy

**Error Handling Emas**: Jangan pernah mengekspos error database ke API response!

1. **Repository Layer**: Mengembalikan error asli (misal `pgx.ErrNoRows`).
2. **Usecase Layer**: Membungkus atau menerjemahkan error eksternal menjadi Sentinel Domain Error (misal `domain.ErrProductNotFound`).
3. **Handler Layer (gRPC)**: Menggunakan fungsi _Mapper_ (seperti `domainErrorToGRPC(err error)`) untuk mengubah `domain.ErrProductNotFound` menjadi `codes.NotFound`.
   
Bila menemukan error `500 Internal Server Error`, _mapper_ harus me-_mask_ (menyembunyikan) detail internal dan hanya menuliskan pesan generik ("Internal server error"), namun tetap me-log error aslinya (beserta Stack Trace dari Recovery Interceptor) ke _Structured Logger_ agar tim _Engineer_ dapat melakukan _debugging_.

Di gRPC, kita bisa melampirkan metadata _Rich Errors_ dari package `google.golang.org/genproto/googleapis/rpc/errdetails`:

```go
st := status.New(codes.InvalidArgument, "Invalid product data")
v := &errdetails.BadRequest_FieldViolation{
    Field:       "PriceCents",
    Description: "Price cannot be negative",
}
br := &errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{v}}
st, _ = st.WithDetails(br)
return st.Err()
```
Ini membuat response yang dikirim ke Frontend (via gRPC-Gateway) berubah menjadi struktur JSON HTTP 400 Bad Request standar yang sangat rapi.

---
# BAGIAN IV: TESTING

## Phase 19: Unit Testing

Di Go, gunakan "Table-Driven Tests". Kita fokus melakukan mocking pada _Repository_ untuk mengetes _Usecase_.

**File**: `services/product-service/internal/usecase/product_usecase_test.go`
```go
package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/usecase"
)

// Buat Mock Repository manual atau generate pakai vektra/mockery
type MockRepo struct {
	mock.Mock
}
func (m *MockRepo) Create(ctx context.Context, p *domain.Product) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
// ... implementasi method lain ...

func TestCreateProduct(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.CreateProductInput
		mockSetup func(m *MockRepo)
		wantError bool
	}{
		{
			name: "Success",
			input: domain.CreateProductInput{Name: "Laptop", PriceCents: 1000},
			mockSetup: func(m *MockRepo) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil)
			},
			wantError: false,
		},
		{
			name: "Validation Error",
			input: domain.CreateProductInput{Name: ""}, // Nama kosong
			mockSetup: func(m *MockRepo) {}, // Repo tidak akan dipanggil
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepo)
			tt.mockSetup(mockRepo)
			
			uc := usecase.NewProductUsecase(mockRepo)
			_, err := uc.Create(context.Background(), tt.input)
			
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}
```

## Phase 20: Integration Testing

Integration test akan menyalakan container PostgreSQL sungguhan menggunakan `testcontainers-go`. Kita pisahkan file ini dengan _build tag_ `//go:build integration`.

**File**: `services/product-service/internal/repository/postgres/product_repo_integration_test.go`
```go
//go:build integration

package postgres_test

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	// ... (setup container logic)
)

func TestProductRepo_Integration(t *testing.T) {
    // 1. Setup Postgres Container
    // 2. Run Migrations (up.sql)
    // 3. Connect pgxpool
    
    // 4. Test CRUD secara end-to-end terhadap DB asli
    t.Run("Create and Get", func(t *testing.T) {
        p := &domain.Product{Name: "Test", PriceCents: 100}
        err := repo.Create(ctx, p)
        assert.NoError(t, err)
        
        fetched, err := repo.GetByID(ctx, p.ID)
        assert.NoError(t, err)
        assert.Equal(t, "Test", fetched.Name)
    })
}
```
Untuk menjalankannya: `go test -tags=integration ./...`

## Phase 21: E2E & Manual Testing

Gunakan `grpcurl` untuk tes manual saat server jalan:
```bash
grpcurl -plaintext -d '{"name": "Baju", "price_cents": 50000, "stock": 10}' \
localhost:50051 product.v1.ProductService/CreateProduct
```
Atau via HTTP (API Gateway):
```bash
curl -X POST http://localhost:8080/v1/products \
-H "Content-Type: application/json" \
-d '{"name": "Baju", "price_cents": 50000, "stock": 10}'
```


---
# BAGIAN V: MULTI-SERVICE & DEPLOYMENT

Setelah menguasai pembuatan satu service yang kokoh, saatnya memperluas sistem. V3 menambahkan _Order Service_ untuk mensimulasikan interaksi _Microservice_ sesungguhnya.

## Phase 22 & 23: Second Service & Inter-Service Communication

Bagaimana `order-service` memanggil `product-service` untuk mengecek stok sebelum membuat pesanan? Kita gunakan gRPC Client.

Di dalam `order-service`, kita membuat layer Infrastructure/Client:

**File**: `services/order-service/internal/infrastructure/grpcclient/product_client.go`
```go
package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
)

type ProductClient interface {
	CheckStock(ctx context.Context, productID string, quantity int32) error
}

type productClient struct {
	client productv1.ProductServiceClient
}

func NewProductClient(conn *grpc.ClientConn) ProductClient {
	return &productClient{
		client: productv1.NewProductServiceClient(conn),
	}
}

func (p *productClient) CheckStock(ctx context.Context, productID string, quantity int32) error {
	resp, err := p.client.GetProduct(ctx, &productv1.GetProductRequest{Id: productID})
	if err != nil {
		return err // Handle gRPC errors (e.g. NotFound)
	}
	
	if resp.Product.Stock < quantity {
		return fmt.Errorf("insufficient stock")
	}
	return nil
}
```

> **Catatan Resiliensi**: Di _production_, panggilan gRPC antar-service harus dilengkapi dengan _Timeout_ (context.WithTimeout), _Circuit Breaker_, dan mekanisme _Retry_ agar kegagalan `product-service` tidak melumpuhkan `order-service` sepenuhnya.

---

## Phase 24: Custom API Gateway (Go)

Daripada setiap service membuka port HTTP sendiri-sendiri, kita membuat satu **API Gateway** menggunakan `grpc-gateway` untuk meneruskan lalu lintas HTTP (JSON) ke berbagai service gRPC di _backend_.

**File**: `services/api-gateway/cmd/server/main.go`
```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	orderv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/order/v1"
)

func main() {
	ctx := context.Background()
	mux := runtime.NewServeMux()
	
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Route ke Product Service
	err := productv1.RegisterProductServiceHandlerFromEndpoint(ctx, mux, "product-service:50051", opts)
	if err != nil {
		log.Fatalf("Fail to register product service: %v", err)
	}

	// Route ke Order Service
	err = orderv1.RegisterOrderServiceHandlerFromEndpoint(ctx, mux, "order-service:50052", opts)
	if err != nil {
		log.Fatalf("Fail to register order service: %v", err)
	}

	// Anda bisa menyelipkan Middleware HTTP murni di sini (CORS, Rate Limiter)
	handler := withCORS(mux)

	log.Println("API Gateway listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// ... handle preflight OPTIONS
		h.ServeHTTP(w, r)
	})
}
```

---

## Phase 25: Docker & Docker Compose

Mari bungkus semua yang sudah kita buat ke dalam _container_.

**File**: `services/product-service/Dockerfile`
```dockerfile
# BUILD STAGE
FROM golang:1.22-alpine AS builder

WORKDIR /app
# Salin go.work dan semua go.mod dari setiap service
COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY services/product-service/go.mod services/product-service/go.sum ./services/product-service/
RUN go mod download

# Salin source code
COPY . .

# Build spesifik product-service
WORKDIR /app/services/product-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server cmd/server/main.go

# RUN STAGE
FROM alpine:3.19
WORKDIR /app

# Non-root user untuk security
RUN adduser -D -g '' appuser
USER appuser

COPY --from=builder /app/services/product-service/server .
COPY --from=builder /app/services/product-service/internal/config/default.yaml ./internal/config/

EXPOSE 50051
CMD ["./server"]
```

**File**: `docker-compose.yml` (Di Root Proyek)
```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: simplestore
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d simplestore"]
      interval: 5s
      timeout: 5s
      retries: 5

  product-service:
    build:
      context: .
      dockerfile: services/product-service/Dockerfile
    environment:
      - DATABASE_URL=postgres://user:password@postgres:5432/simplestore?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "50051:50051"

  order-service:
    build:
      context: .
      dockerfile: services/order-service/Dockerfile
    environment:
      - DATABASE_URL=postgres://user:password@postgres:5432/simplestore?sslmode=disable
    depends_on:
      product-service:
        condition: service_started

  api-gateway:
    build:
      context: .
      dockerfile: services/api-gateway/Dockerfile
    ports:
      - "8080:8080"
    depends_on:
      - product-service
      - order-service
```
Jalankan dengan: `docker compose up -d --build`. Seluruh sistem microservice Anda sekarang menyala!

---

## Phase 26 & 27: CI/CD & Kubernetes Basics

**CI/CD Pipeline (GitHub Actions)**:
Buat file `.github/workflows/ci.yml`. Pipeline minimal harus memiliki jobs:
1. `golangci-lint` (Memastikan kode bersih dari anti-pattern)
2. `go test -race` (Menjalankan unit test)
3. `docker build` (Memastikan image bisa di-build dengan sukses)

**Kubernetes Basics**:
Untuk deployment ke production K8s, gunakan:
- **Deployment**: Untuk menjalankan replica service Anda.
- **Service (ClusterIP)**: Untuk Load Balancing internal antar-pod (misal: Gateway ke Product Service).
- **ConfigMap & Secret**: Untuk menyuntikkan konfigurasi dan password database tanpa melakukan _hardcode_.
- Gunakan _Liveness Probe_ yang mengarah ke port gRPC Health Check yang telah kita buat di Phase 15.

---

# BAGIAN VI: REFERENSI (APPENDIX)

## Appendix A: Kenapa Bukan GORM?
GORM sangat bagus untuk prototyping, tetapi dalam _Clean Architecture_ berskala produksi, ia menjadi beban:
1. **Coupling**: Domain entitas terkotori oleh tag `gorm:"primaryKey"`. Domain seharusnya tidak peduli jenis DB apa yang digunakan.
2. **Magic**: Lazy loading dan auto-save sering menyebabkan "N+1 query problem" yang membunuh performa API Anda secara diam-diam.
3. **Kendali**: `pgx` memberikan kontrol penuh atas raw SQL, memastikan kita bisa mengoptimalkan query dengan index atau tipe data PostgreSQL yang spesifik.

## Appendix B: Kenapa Bukan Viper?
Viper adalah standar de-facto di Go masa lalu. Namun, pola `viper.GetString("DATABASE_URL")` yang tersebar di berbagai layer adalah **Global State Anti-Pattern**. Jika sebuah Usecase memanggil Viper secara global, Anda tidak bisa melakukan Unit Test dengan mudah. `koanf` lebih ringan dan memaksa kita menggunakan struct yang di-inject (DI), sehingga lebih bersih.

## Appendix C: Kenapa Bukan Zap/Logrus?
Logrus sudah dalam status "maintenance mode". Uber's Zap sangat cepat, tetapi sejak Go 1.21 merilis `log/slog` bawaan (standard library), performanya sudah sangat cukup untuk 99% aplikasi production. Menggunakan `slog` mengurangi jumlah dependensi external proyek Anda.

## Appendix D: Rangkuman Flow Gagal (Error Handling)
1. **Repo**: Mengembalikan error asli (misal `pgx.ErrNoRows`).
2. **Usecase**: Jika menerima `ErrNoRows`, bungkus/kembalikan sebagai Sentinel Domain Error (misal `domain.ErrProductNotFound`).
3. **Handler**: Panggil `domainErrorToGRPC()`. Mapper ini akan mengubah `ErrProductNotFound` menjadi gRPC Status Code `codes.NotFound` (yang otomatis diterjemahkan grpc-gateway menjadi HTTP 404). Ujung client menerima error JSON yang rapi, tanpa bocoran detail SQL.

## 🏆 Kesimpulan & "What's Next"

Selamat! Anda telah memahami bagaimana microservice Go di industri nyata dirancang. Alur _Clean Architecture_, komunikasi aman gRPC, interceptor, hingga konfigurasi monorepo memberikan pondasi kokoh.

**Langkah selanjutnya untuk eksplorasi mandiri:**
- Event-Driven Architecture (Kirim event ke Kafka/RabbitMQ setelah _CreateOrder_ berhasil).
- Distributed Tracing (Sambungkan OpenTelemetry ke Jaeger/Datadog).
- Helm Charts (Template Kubernetes deployment).
