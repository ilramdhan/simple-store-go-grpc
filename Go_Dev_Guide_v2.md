# 📘 Go gRPC Development Guide
### `simple-store-go-grpc` — Panduan Lengkap dari Awal sampai Selesai

> **Untuk siapa**: Developer yang ingin belajar membangun Go microservice dengan gRPC, Clean Architecture, dan tooling modern.
> **Repo**: `github.com/ilramdhan/simple-store-go-grpc`

---

## 🗺️ Peta Perjalanan (Urutan Wajib)

```
Phase 1  → Inisialisasi Monorepo & Tools
Phase 2  → Definisi Proto (Schema-First API)
Phase 3  → Generate Kode dari Proto          ← batas panduan lama
Phase 4  → Domain Layer (Entitas & Interface)
Phase 5  → Config Layer (Konfigurasi Aplikasi)
Phase 6  → Repository Layer (Database)
Phase 7  → Usecase Layer (Business Logic)
Phase 8  → Handler Layer (gRPC Transport)
Phase 9  → Server Wiring (Merangkai Semuanya)
Phase 10 → Testing dengan Postman / grpcurl
Phase 11 → Docker & docker-compose
```

> **Mengapa urutan ini?**
> Domain Layer harus dibuat **sebelum** Repository dan Usecase, karena keduanya bergantung pada
> interface dan entitas yang didefinisikan di Domain. Ini adalah inti dari Clean Architecture:
> lapisan luar bergantung ke dalam, bukan sebaliknya.

---

## Phase 1: Inisialisasi Monorepo & Tools

### 1.1 Prasyarat — Install Tools

```bash
# Pastikan Go sudah terinstall (cek versi)
go version
# Harus: go1.26.x atau lebih baru

# Install Buf CLI
# Untuk Linux (pilih salah satu):
curl -sSL https://github.com/bufbuild/buf/releases/latest/download/buf-Linux-x86_64 \
  -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf
# Atau via Go:
go install github.com/bufbuild/buf/cmd/buf@latest

# Verifikasi
buf --version  # Harus: 1.65.0 atau lebih baru

# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin latest
golangci-lint --version
```

### 1.2 Struktur Monorepo yang Akan Kita Bangun

Clone repo:
```bash
git clone https://github.com/ilramdhan/simple-store-go-grpc.git
cd simple-store-go-grpc
```

Struktur lengkap:
```
simple-store-go-grpc/                    ← Root monorepo
│
├── proto/                               ← Sumber kebenaran API
│   ├── common/v1/common.proto           ← Shared types (BaseResponse, dll.)
│   └── product/v1/product.proto         ← ProductService
│
├── gen/                                 ← ⚠️ AUTO-GENERATED, jangan edit!
│   ├── go/common/v1/common.pb.go
│   ├── go/product/v1/product.pb.go
│   ├── go/product/v1/product_grpc.pb.go
│   ├── go/product/v1/product.pb.gw.go
│   └── openapiv2/api.swagger.json
│
├── services/
│   └── product-service/                 ← Service mandiri (Go module sendiri)
│       ├── cmd/server/main.go           ← Entry point (hanya wiring)
│       ├── internal/
│       │   ├── config/config.go         ← Konfigurasi dari env vars
│       │   ├── domain/                  ← INTI: entitas + interfaces
│       │   │   ├── product.go
│       │   │   └── errors.go
│       │   ├── usecase/                 ← Business logic
│       │   │   ├── product_usecase.go
│       │   │   └── response_builder.go
│       │   ├── repository/
│       │   │   ├── postgres/product_repo.go
│       │   │   └── migrations/
│       │   │       ├── 000001_create_products.up.sql
│       │   │       └── 000001_create_products.down.sql
│       │   └── handler/grpc/
│       │       └── product_handler.go
│       ├── go.mod
│       └── go.sum
│
├── buf.yaml              ← Buf workspace config (v2)
├── buf.gen.yaml          ← Buf code generation config (v2)
├── buf.lock              ← Pinned buf dependencies
├── Makefile
├── .gitignore
└── README.md
```

### 1.3 Root Module (sudah ada)

```go
// File: go.mod (di ROOT, bukan di dalam services/)
module github.com/ilramdhan/simple-store-go-grpc

go 1.26.0
```

> ⚠️ Root module ini hanya untuk proto/gen.
> Setiap service punya `go.mod` sendiri di dalam foldernya.

---

## Phase 2: Definisi Proto (Schema-First API)

### Apa itu Proto dan kenapa Schema-First?

**Protocol Buffers (proto)** adalah bahasa untuk mendefinisikan API — seperti "kontrak" antara
client dan server. Dari satu file `.proto`, kita generate:
- Struct Go untuk data types
- Interface gRPC server/client
- HTTP handler (REST via grpc-gateway)
- Dokumentasi Swagger/OpenAPI

**Schema-First** = tulis proto dulu, baru implementasi. API terdefinisi jelas sebelum ada kode.

### 2.1 Common Proto

**File**: `proto/common/v1/common.proto`

```protobuf
syntax = "proto3";

package common.v1;

import "google/protobuf/timestamp.proto";

// BaseResponse adalah pembungkus standar untuk SEMUA response.
// Setiap RPC mengembalikan ini sehingga client selalu tahu struktur errornya.
message BaseResponse {
  int32 http_status_code = 1;  // Mirroring HTTP: 200, 201, 400, 404, 500
  bool  is_success       = 2;  // true jika operasi berhasil
  string message         = 3;  // Pesan human-readable
  repeated ValidationError validation_errors = 4;  // Diisi jika status 400
}

// ValidationError mendeskripsikan satu field yang gagal validasi.
message ValidationError {
  string field   = 1;  // Nama field yang bermasalah
  string message = 2;  // Alasan kenapa gagal
}

// PaginationRequest untuk request list dengan halaman.
message PaginationRequest {
  int32 page      = 1;  // Halaman ke berapa (mulai dari 1)
  int32 page_size = 2;  // Jumlah item per halaman (max 100)
}

// PaginationResponse metadata hasil list.
message PaginationResponse {
  int32 current_page = 1;
  int32 page_size    = 2;
  int64 total_items  = 3;
  int32 total_pages  = 4;
}

// AuditInfo metadata audit standar.
// Gunakan google.protobuf.Timestamp (BUKAN string!) agar type-safe dan timezone-aware.
message AuditInfo {
  google.protobuf.Timestamp created_at = 1;
  string                    created_by = 2;
  google.protobuf.Timestamp updated_at = 3;
  string                    updated_by = 4;
}
```

> **Kenapa `google.protobuf.Timestamp` bukan `string`?**
> String tidak punya format standar (ISO 8601? Unix? timezone apa?).
> `Timestamp` memastikan semua timestamp dalam format RFC 3339 (UTC) secara otomatis.

### 2.2 Product Proto

**File**: `proto/product/v1/product.proto`

```protobuf
syntax = "proto3";

package product.v1;

import "common/v1/common.proto";
import "google/api/annotations.proto";
import "google/protobuf/field_mask.proto";

// Product adalah entitas utama dalam katalog produk.
message Product {
  string id          = 1;
  string name        = 2;
  string description = 3;
  // price_cents menyimpan harga dalam satuan terkecil (sen/rupiah)
  // Contoh: 1999 = Rp 19.99 atau $19.99
  // WAJIB integer! Floating point (double) TIDAK AMAN untuk uang.
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
  string id = 1;  // UUID v4
}

message GetProductResponse {
  common.v1.BaseResponse base    = 1;
  Product                product = 2;
}

// UpdateProductRequest menggunakan FieldMask untuk partial update.
// Contoh: update_mask = "name,stock" → hanya name & stock yang berubah.
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
  repeated Product             products   = 2;  // PLURAL! bukan product
  common.v1.PaginationResponse pagination = 3;
}

// ProductService mendefinisikan semua operasi CRUD.
// Setiap RPC punya HTTP annotation untuk grpc-gateway (REST).
service ProductService {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse) {
    option (google.api.http) = {
      post: "/v1/products"  // WAJIB ada leading slash /
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

---

## Phase 3: Generate Kode dari Proto

### 3.1 Konfigurasi Buf (sudah ada)

**File**: `buf.yaml`
```yaml
version: v2                  # WAJIB v2, bukan v1!
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

**File**: `buf.gen.yaml`
```yaml
version: v2
managed:
  enabled: true
  override:
    # managed mode: go_package di-generate otomatis, tidak perlu tulis manual di .proto
    - file_option: go_package_prefix
      value: github.com/ilramdhan/simple-store-go-grpc/gen/go
plugins:
  # Plugin 1: Generate struct Go dari proto messages
  - remote: buf.build/protocolbuffers/go:v1.36.11
    out: gen/go
    opt: [paths=source_relative]

  # Plugin 2: Generate gRPC client/server interface
  - remote: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
      - require_unimplemented_servers=true

  # Plugin 3: Generate HTTP handler (grpc-gateway = akses via REST)
  - remote: buf.build/grpc-ecosystem/gateway:v2.29.0
    out: gen/go
    opt:
      - paths=source_relative
      - generate_unbound_methods=true

  # Plugin 4: Generate dokumentasi Swagger/OpenAPI
  - remote: buf.build/grpc-ecosystem/openapiv2:v2.29.0
    out: gen/openapiv2
    opt:
      - generate_unbound_methods=true
      - allow_merge=true
      - merge_file_name=api
inputs:
  - directory: proto
```

### 3.2 Jalankan Generate

```bash
# Di root monorepo!

# Step 1: Download/update dependencies buf
make proto-dep-update
# Atau langsung: buf dep update

# Step 2: Lint proto (pastikan tidak ada error)
make proto-lint
# Atau: buf lint
# Output KOSONG = tidak ada error ✅

# Step 3: Generate semua kode
make proto-generate
# Atau: buf generate
```

### 3.3 Verifikasi Hasil Generate

```bash
find gen/ -type f | sort

# Output yang diharapkan:
# gen/go/common/v1/common.pb.go        ← Struct Go dari common.proto
# gen/go/product/v1/product.pb.go      ← Struct Go dari product.proto
# gen/go/product/v1/product_grpc.pb.go ← Interface gRPC server & client
# gen/go/product/v1/product.pb.gw.go   ← HTTP handler (grpc-gateway)
# gen/openapiv2/api.swagger.json        ← Dokumentasi Swagger
```

> ⚠️ **JANGAN edit file di `gen/`!** Mereka akan tertimpa setiap kali `buf generate` dijalankan.
> Jika perlu mengubah sesuatu, ubah di file `.proto` lalu generate ulang.

### 3.4 Memahami File yang Di-generate

**`product_grpc.pb.go`** — berisi interface yang HARUS diimplementasikan oleh Handler:
```go
// Interface ini yang harus kamu implementasikan di Handler layer.
// Dibuat otomatis dari service ProductService di product.proto.
type ProductServiceServer interface {
    CreateProduct(context.Context, *CreateProductRequest) (*CreateProductResponse, error)
    GetProduct(context.Context, *GetProductRequest) (*GetProductResponse, error)
    UpdateProduct(context.Context, *UpdateProductRequest) (*UpdateProductResponse, error)
    DeleteProduct(context.Context, *DeleteProductRequest) (*DeleteProductResponse, error)
    ListProducts(context.Context, *ListProductsRequest) (*ListProductsResponse, error)
}
```

---

## Phase 4: Domain Layer

### Apa itu Domain Layer dan mengapa penting?

**Domain Layer** adalah **inti dari aplikasi** — tempat mendefinisikan:
1. **Entitas (Entity)**: Representasi bisnis dari objek yang kita kelola
2. **Repository Interface**: "Kontrak" tentang bagaimana data disimpan, tapi tidak peduli implementasinya

> **Analogi**: Bayangkan sebuah toko. Domain adalah deskripsi produk di katalog:
> "Produk punya nama, harga, dan stok." Toko tidak peduli apakah produk disimpan di rak A
> atau rak B — itu urusan bagian gudang (Repository).

**Aturan penting**: Domain Layer **TIDAK BOLEH** mengimport package dari layer luar
(database driver, gRPC, HTTP). Hanya boleh menggunakan standard library Go.

### 4.1 Buat Direktori

```bash
cd services/product-service
mkdir -p internal/domain
```

### 4.2 Domain Entity & Repository Interface

**File**: `services/product-service/internal/domain/product.go`

```go
package domain

import (
    "context"
    "time"
)

// Product adalah entitas domain — representasi bisnis dari sebuah produk.
// Ini BUKAN struct dari proto (gen/go/) dan BUKAN struct dari database.
// Ini adalah "bahasa bisnis" yang dimengerti semua layer.
type Product struct {
    ID          string
    Name        string
    Description string
    // PriceCents menyimpan harga dalam satuan terkecil.
    // Gunakan integer! Floating point tidak akurat untuk uang.
    // Contoh: 19999 = Rp 199.99
    PriceCents  int64
    Stock       int32
    CreatedAt   time.Time
    UpdatedAt   time.Time
    CreatedBy   string
    UpdatedBy   string
}

// ProductRepository adalah "kontrak" tentang operasi database yang dibutuhkan.
//
// Interface ini DIDEFINISIKAN di Domain, tapi DIIMPLEMENTASIKAN di Repository layer.
// Inilah Dependency Inversion Principle (DIP).
//
// Kenapa interface bukan langsung implementasi?
// → Bisa ganti database (PostgreSQL → MySQL) tanpa mengubah business logic.
// → Bisa pakai mock/fake di unit test tanpa database nyata.
type ProductRepository interface {
    // Create menyimpan produk baru ke database.
    Create(ctx context.Context, p *Product) error

    // GetByID mengambil produk berdasarkan ID.
    // Mengembalikan (nil, ErrProductNotFound) jika tidak ditemukan.
    GetByID(ctx context.Context, id string) (*Product, error)

    // Update memperbarui produk yang sudah ada.
    Update(ctx context.Context, p *Product) error

    // Delete menghapus produk berdasarkan ID (soft delete).
    Delete(ctx context.Context, id string) error

    // List mengambil daftar produk dengan paginasi.
    // Return: slice produk, total semua produk, error
    List(ctx context.Context, page, pageSize int32) ([]*Product, int64, error)
}
```

### 4.3 Domain Errors

**File**: `services/product-service/internal/domain/errors.go`

```go
package domain

import "errors"

// Error domain khusus untuk kondisi bisnis.
// Layer Handler akan mengecek jenis error ini untuk menentukan status HTTP.

// ErrProductNotFound dikembalikan ketika produk tidak ada di database.
var ErrProductNotFound = errors.New("product not found")

// ErrProductNameEmpty dikembalikan ketika nama produk kosong.
var ErrProductNameEmpty = errors.New("product name cannot be empty")

// ErrProductPriceNegative dikembalikan ketika harga negatif.
var ErrProductPriceNegative = errors.New("product price cannot be negative")

// ErrProductStockNegative dikembalikan ketika stok negatif.
var ErrProductStockNegative = errors.New("product stock cannot be negative")
```

> **Kenapa error domain terpisah?**
> Layer Usecase bisa melakukan `errors.Is(err, domain.ErrProductNotFound)` untuk
> membedakan "produk tidak ada" (→ 404) dari "database error" (→ 500).

---

## Phase 5: Config Layer

### Apa itu Config Layer?

Config layer membaca **konfigurasi aplikasi dari environment variables**. Best practice ini disebut
[12-Factor App](https://12factor.net/) dan memastikan:
- Password/API key tidak hardcode di kode
- Mudah di-deploy di berbagai environment (dev, staging, production)

### 5.1 Install Dependencies

```bash
# Di dalam services/product-service/
cd services/product-service

go get github.com/caarlos0/env/v11          # Baca env vars ke struct
go get github.com/jackc/pgx/v5             # PostgreSQL driver (performa terbaik)
go get github.com/jackc/pgx/v5/pgxpool     # Connection pooling
go get github.com/google/uuid              # Generate UUID
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/go-playground/validator/v10
go get google.golang.org/grpc
go get google.golang.org/protobuf
go get github.com/grpc-ecosystem/grpc-gateway/v2

go mod tidy  # Bersihkan go.mod
```

### 5.2 Buat Config Struct

```bash
mkdir -p internal/config
```

**File**: `services/product-service/internal/config/config.go`

```go
package config

import (
    "log/slog"
    "os"

    "github.com/caarlos0/env/v11"
)

// Config menyimpan semua konfigurasi dari environment variables.
// Tag `env:"..."` = nama env var, `envDefault:"..."` = nilai default.
type Config struct {
    GRPCPort    string `env:"GRPC_PORT"    envDefault:"50051"`
    HTTPPort    string `env:"HTTP_PORT"    envDefault:"8080"`
    DatabaseURL string `env:"DATABASE_URL" envRequired:"true"`
    AppEnv      string `env:"APP_ENV"      envDefault:"development"`
    LogLevel    string `env:"LOG_LEVEL"    envDefault:"info"`
}

// Load membaca environment variables dan mengembalikan Config.
// Akan exit(1) jika ada yang required tapi tidak ada.
func Load() *Config {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }
    return cfg
}
```

### 5.3 Buat File `.env` untuk Development

```bash
# Di services/product-service/ — JANGAN commit file ini!
cat > .env << 'EOF'
GRPC_PORT=50051
HTTP_PORT=8080
DATABASE_URL=postgres://storeuser:storepass@localhost:5432/storedb?sslmode=disable
APP_ENV=development
LOG_LEVEL=debug
EOF
```

---

## Phase 6: Repository Layer (Database)

### Apa itu Repository Layer?

**Repository Layer** adalah implementasi konkret dari `ProductRepository` interface yang
didefinisikan di Domain. Layer ini berinteraksi langsung dengan database.

> **Analogi lanjutan**: Jika Domain adalah katalog buku dengan aturan "buku harus punya judul",
> Repository adalah staf gudang yang benar-benar mengambil buku dari rak.

**Aturan**: Repository hanya boleh bergantung pada `domain` package.
Tidak boleh bergantung pada `usecase` atau `handler`.

### 6.1 Buat Direktori

```bash
mkdir -p internal/repository/postgres
mkdir -p internal/repository/migrations
```

### 6.2 SQL Migration

Migration adalah script SQL yang mendefinisikan perubahan schema database secara terurut.
Setiap migration punya versi (000001, 000002, ...) dan bisa di-rollback.

**File**: `internal/repository/migrations/000001_create_products.up.sql`
```sql
-- Script ini dijalankan saat "migrate up" (membuat tabel)
CREATE TABLE IF NOT EXISTS products (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    -- Harga disimpan sebagai integer (sen), BUKAN REAL/FLOAT!
    price_cents BIGINT       NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    stock       INTEGER      NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(255) NOT NULL DEFAULT 'system',
    updated_by  VARCHAR(255) NOT NULL DEFAULT 'system',
    -- NULL berarti produk aktif; tidak NULL berarti sudah dihapus (soft delete)
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_products_deleted_at  ON products(deleted_at);
CREATE INDEX IF NOT EXISTS idx_products_created_at  ON products(created_at DESC);
```

**File**: `internal/repository/migrations/000001_create_products.down.sql`
```sql
-- Script ini dijalankan saat "migrate down" (rollback)
DROP TABLE IF EXISTS products;
```

> **Kenapa UUID bukan auto-increment?** UUID lebih aman (tidak bisa ditebak urutannya) dan
> cocok untuk sistem terdistribusi di mana beberapa server mungkin generate ID secara bersamaan.

> **Kenapa `deleted_at` (soft delete)?** Data yang dihapus masih tersimpan untuk audit trail.
> Query cukup tambahkan `WHERE deleted_at IS NULL`.

### 6.3 Implementasi Repository

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

// ProductRepo adalah implementasi konkret dari domain.ProductRepository.
// Menggunakan pgxpool untuk connection pooling (lebih efisien dari koneksi tunggal).
type ProductRepo struct {
    db *pgxpool.Pool
}

// New membuat instance baru ProductRepo.
func New(db *pgxpool.Pool) *ProductRepo {
    return &ProductRepo{db: db}
}

// Create menyimpan produk baru ke database.
func (r *ProductRepo) Create(ctx context.Context, p *domain.Product) error {
    query := `
        INSERT INTO products (id, name, description, price_cents, stock, created_by, updated_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING created_at, updated_at
    `
    // Gunakan parameterized query ($1, $2, ...) SELALU — mencegah SQL Injection!
    err := r.db.QueryRow(ctx, query,
        p.ID, p.Name, p.Description, p.PriceCents, p.Stock, p.CreatedBy, p.UpdatedBy,
    ).Scan(&p.CreatedAt, &p.UpdatedAt)

    if err != nil {
        return fmt.Errorf("ProductRepo.Create: %w", err)
    }
    return nil
}

// GetByID mengambil produk berdasarkan ID.
func (r *ProductRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
    query := `
        SELECT id, name, description, price_cents, stock,
               created_at, updated_at, created_by, updated_by
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
            // Kembalikan error domain, bukan error database mentah
            return nil, domain.ErrProductNotFound
        }
        return nil, fmt.Errorf("ProductRepo.GetByID: %w", err)
    }
    return p, nil
}

// Update memperbarui data produk.
func (r *ProductRepo) Update(ctx context.Context, p *domain.Product) error {
    query := `
        UPDATE products
        SET name = $1, description = $2, price_cents = $3,
            stock = $4, updated_at = NOW(), updated_by = $5
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
        return fmt.Errorf("ProductRepo.Update: %w", err)
    }
    return nil
}

// Delete menghapus produk secara soft-delete (set deleted_at, data tetap ada di DB).
func (r *ProductRepo) Delete(ctx context.Context, id string) error {
    query := `
        UPDATE products SET deleted_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
    result, err := r.db.Exec(ctx, query, id)
    if err != nil {
        return fmt.Errorf("ProductRepo.Delete: %w", err)
    }
    if result.RowsAffected() == 0 {
        return domain.ErrProductNotFound
    }
    return nil
}

// List mengambil daftar produk dengan paginasi.
func (r *ProductRepo) List(ctx context.Context, page, pageSize int32) ([]*domain.Product, int64, error) {
    // Hitung offset: page=2, pageSize=10 → skip 10 item pertama
    offset := (page - 1) * pageSize

    // Hitung total terlebih dahulu (untuk metadata pagination)
    var total int64
    if err := r.db.QueryRow(ctx,
        `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`,
    ).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("ProductRepo.List count: %w", err)
    }

    // Ambil data dengan LIMIT dan OFFSET
    rows, err := r.db.Query(ctx, `
        SELECT id, name, description, price_cents, stock,
               created_at, updated_at, created_by, updated_by
        FROM products
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `, pageSize, offset)
    if err != nil {
        return nil, 0, fmt.Errorf("ProductRepo.List query: %w", err)
    }
    defer rows.Close()

    var products []*domain.Product
    for rows.Next() {
        p := &domain.Product{}
        if err := rows.Scan(
            &p.ID, &p.Name, &p.Description, &p.PriceCents, &p.Stock,
            &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
        ); err != nil {
            return nil, 0, fmt.Errorf("ProductRepo.List scan: %w", err)
        }
        products = append(products, p)
    }
    return products, total, nil
}
```

---

## Phase 7: Usecase Layer (Business Logic)

### Apa itu Usecase Layer?

**Usecase Layer** adalah tempat semua **logika bisnis** berada — "otak" aplikasi.

Usecase bertugas:
1. Menerima request (dari Handler)
2. Menjalankan aturan bisnis (validasi, kalkulasi, dll.)
3. Memanggil Repository untuk operasi database
4. Membangun response

> **Analogi**: Di toko, Usecase adalah kasir yang menjalankan prosedur: "Cek stok → kurangi
> stok → cetak struk → konfirmasi pembelian."

**Aturan**: Usecase bergantung pada `domain` interface. Tidak bergantung pada
implementasi konkret Repository, tidak pada gRPC, tidak pada HTTP.

### 7.1 Response Builder Helper

**File**: `services/product-service/internal/usecase/response_builder.go`

```go
package usecase

import (
    "net/http"

    commonv1  "github.com/ilramdhan/simple-store-go-grpc/gen/go/common/v1"
    productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
    "google.golang.org/protobuf/types/known/timestamppb"

    "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

// Fungsi-fungsi berikut membangun BaseResponse sesuai kondisi.
// Dipakai di semua method usecase agar tidak ada pengulangan kode.

func successResponse(msg string) *commonv1.BaseResponse {
    return &commonv1.BaseResponse{
        HttpStatusCode: http.StatusOK,
        IsSuccess:      true,
        Message:        msg,
    }
}

func createdResponse(msg string) *commonv1.BaseResponse {
    return &commonv1.BaseResponse{
        HttpStatusCode: http.StatusCreated,
        IsSuccess:      true,
        Message:        msg,
    }
}

func notFoundResponse(msg string) *commonv1.BaseResponse {
    return &commonv1.BaseResponse{
        HttpStatusCode: http.StatusNotFound,
        IsSuccess:      false,
        Message:        msg,
    }
}

func internalErrorResponse(msg string) *commonv1.BaseResponse {
    return &commonv1.BaseResponse{
        HttpStatusCode: http.StatusInternalServerError,
        IsSuccess:      false,
        Message:        msg,
    }
}

func validationErrorResponse(errs []*commonv1.ValidationError) *commonv1.BaseResponse {
    return &commonv1.BaseResponse{
        HttpStatusCode:   http.StatusBadRequest,
        IsSuccess:        false,
        Message:          "Validation failed",
        ValidationErrors: errs,
    }
}

// domainToProtoProduct mengkonversi domain.Product ke proto Product.
// Ini ada di usecase karena usecase yang "tahu" tentang keduanya (domain + proto).
func domainToProtoProduct(p *domain.Product) *productv1.Product {
    return &productv1.Product{
        Id:          p.ID,
        Name:        p.Name,
        Description: p.Description,
        PriceCents:  p.PriceCents,
        Stock:       p.Stock,
        Audit: &commonv1.AuditInfo{
            // time.Time → google.protobuf.Timestamp menggunakan timestamppb.New()
            CreatedAt: timestamppb.New(p.CreatedAt),
            UpdatedAt: timestamppb.New(p.UpdatedAt),
            CreatedBy: p.CreatedBy,
            UpdatedBy: p.UpdatedBy,
        },
    }
}
```

### 7.2 Product Usecase

**File**: `services/product-service/internal/usecase/product_usecase.go`

```go
package usecase

import (
    "context"
    "errors"
    "log/slog"
    "math"

    "github.com/go-playground/validator/v10"
    "github.com/google/uuid"

    commonv1  "github.com/ilramdhan/simple-store-go-grpc/gen/go/common/v1"
    productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
    "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

// ProductUsecase berisi semua business logic untuk produk.
type ProductUsecase struct {
    repo      domain.ProductRepository  // Interface! bukan implementasi konkret
    validator *validator.Validate
}

// New membuat instance ProductUsecase baru.
func New(repo domain.ProductRepository) *ProductUsecase {
    return &ProductUsecase{
        repo:      repo,
        validator: validator.New(),
    }
}

// ─── CREATE ───────────────────────────────────────────────────────────

// Struct bantu untuk validasi input — proto struct tidak bisa diberi tag validate.
type createProductInput struct {
    Name        string `validate:"required,min=1,max=255"`
    Description string `validate:"max=2000"`
    PriceCents  int64  `validate:"min=0"`
    Stock       int32  `validate:"min=0"`
}

func (uc *ProductUsecase) CreateProduct(
    ctx context.Context,
    req *productv1.CreateProductRequest,
) (*productv1.CreateProductResponse, error) {

    // 1. Validasi input
    input := createProductInput{
        Name:        req.GetName(),
        Description: req.GetDescription(),
        PriceCents:  req.GetPriceCents(),
        Stock:       req.GetStock(),
    }
    if errs := uc.validateInput(input); errs != nil {
        return &productv1.CreateProductResponse{Base: validationErrorResponse(errs)}, nil
    }

    // 2. Buat domain object
    p := &domain.Product{
        ID:          uuid.New().String(),
        Name:        req.GetName(),
        Description: req.GetDescription(),
        PriceCents:  req.GetPriceCents(),
        Stock:       req.GetStock(),
        CreatedBy:   "system",  // Nanti dari JWT auth context
        UpdatedBy:   "system",
    }

    // 3. Simpan ke database
    if err := uc.repo.Create(ctx, p); err != nil {
        slog.Error("CreateProduct: repo error", "error", err)
        return &productv1.CreateProductResponse{Base: internalErrorResponse("Failed to create product")}, nil
    }

    // 4. Return response
    return &productv1.CreateProductResponse{
        Base:    createdResponse("Product created successfully"),
        Product: domainToProtoProduct(p),
    }, nil
}

// ─── GET ──────────────────────────────────────────────────────────────

func (uc *ProductUsecase) GetProduct(
    ctx context.Context,
    req *productv1.GetProductRequest,
) (*productv1.GetProductResponse, error) {

    if req.GetId() == "" {
        return &productv1.GetProductResponse{
            Base: validationErrorResponse([]*commonv1.ValidationError{
                {Field: "id", Message: "id is required"},
            }),
        }, nil
    }

    p, err := uc.repo.GetByID(ctx, req.GetId())
    if err != nil {
        if errors.Is(err, domain.ErrProductNotFound) {
            return &productv1.GetProductResponse{Base: notFoundResponse("Product not found")}, nil
        }
        slog.Error("GetProduct: repo error", "id", req.GetId(), "error", err)
        return &productv1.GetProductResponse{Base: internalErrorResponse("Failed to retrieve product")}, nil
    }

    return &productv1.GetProductResponse{
        Base:    successResponse("Product retrieved successfully"),
        Product: domainToProtoProduct(p),
    }, nil
}

// ─── UPDATE (dengan FieldMask) ────────────────────────────────────────

func (uc *ProductUsecase) UpdateProduct(
    ctx context.Context,
    req *productv1.UpdateProductRequest,
) (*productv1.UpdateProductResponse, error) {

    if req.GetId() == "" {
        return &productv1.UpdateProductResponse{
            Base: validationErrorResponse([]*commonv1.ValidationError{
                {Field: "id", Message: "id is required"},
            }),
        }, nil
    }

    // Ambil data yang ada
    existing, err := uc.repo.GetByID(ctx, req.GetId())
    if err != nil {
        if errors.Is(err, domain.ErrProductNotFound) {
            return &productv1.UpdateProductResponse{Base: notFoundResponse("Product not found")}, nil
        }
        return &productv1.UpdateProductResponse{Base: internalErrorResponse("Failed to retrieve product")}, nil
    }

    // Terapkan FieldMask: hanya update field yang disebutkan
    mask := req.GetUpdateMask()
    if mask == nil || len(mask.GetPaths()) == 0 {
        // Tidak ada mask → update semua field
        if req.GetProduct() != nil {
            existing.Name        = req.GetProduct().GetName()
            existing.Description = req.GetProduct().GetDescription()
            existing.PriceCents  = req.GetProduct().GetPriceCents()
            existing.Stock       = req.GetProduct().GetStock()
        }
    } else {
        // Ada mask → hanya update field yang disebutkan
        for _, path := range mask.GetPaths() {
            switch path {
            case "name":
                existing.Name = req.GetProduct().GetName()
            case "description":
                existing.Description = req.GetProduct().GetDescription()
            case "price_cents":
                existing.PriceCents = req.GetProduct().GetPriceCents()
            case "stock":
                existing.Stock = req.GetProduct().GetStock()
            }
        }
    }
    existing.UpdatedBy = "system"

    if err := uc.repo.Update(ctx, existing); err != nil {
        if errors.Is(err, domain.ErrProductNotFound) {
            return &productv1.UpdateProductResponse{Base: notFoundResponse("Product not found")}, nil
        }
        slog.Error("UpdateProduct: repo error", "id", req.GetId(), "error", err)
        return &productv1.UpdateProductResponse{Base: internalErrorResponse("Failed to update product")}, nil
    }

    return &productv1.UpdateProductResponse{
        Base:    successResponse("Product updated successfully"),
        Product: domainToProtoProduct(existing),
    }, nil
}

// ─── DELETE ───────────────────────────────────────────────────────────

func (uc *ProductUsecase) DeleteProduct(
    ctx context.Context,
    req *productv1.DeleteProductRequest,
) (*productv1.DeleteProductResponse, error) {

    if req.GetId() == "" {
        return &productv1.DeleteProductResponse{
            Base: validationErrorResponse([]*commonv1.ValidationError{
                {Field: "id", Message: "id is required"},
            }),
        }, nil
    }

    if err := uc.repo.Delete(ctx, req.GetId()); err != nil {
        if errors.Is(err, domain.ErrProductNotFound) {
            return &productv1.DeleteProductResponse{Base: notFoundResponse("Product not found")}, nil
        }
        slog.Error("DeleteProduct: repo error", "id", req.GetId(), "error", err)
        return &productv1.DeleteProductResponse{Base: internalErrorResponse("Failed to delete product")}, nil
    }

    return &productv1.DeleteProductResponse{
        Base: successResponse("Product deleted successfully"),
    }, nil
}

// ─── LIST ─────────────────────────────────────────────────────────────

func (uc *ProductUsecase) ListProducts(
    ctx context.Context,
    req *productv1.ListProductsRequest,
) (*productv1.ListProductsResponse, error) {

    page     := req.GetPagination().GetPage()
    pageSize := req.GetPagination().GetPageSize()

    // Terapkan default dan batas
    if page <= 0 { page = 1 }
    if pageSize <= 0 || pageSize > 100 { pageSize = 10 }

    products, total, err := uc.repo.List(ctx, page, pageSize)
    if err != nil {
        slog.Error("ListProducts: repo error", "error", err)
        return &productv1.ListProductsResponse{Base: internalErrorResponse("Failed to retrieve products")}, nil
    }

    // Konversi domain → proto
    protoProducts := make([]*productv1.Product, 0, len(products))
    for _, p := range products {
        protoProducts = append(protoProducts, domainToProtoProduct(p))
    }

    totalPages := int32(math.Ceil(float64(total) / float64(pageSize)))

    return &productv1.ListProductsResponse{
        Base:     successResponse("Products retrieved successfully"),
        Products: protoProducts,
        Pagination: &commonv1.PaginationResponse{
            CurrentPage: page,
            PageSize:    pageSize,
            TotalItems:  total,
            TotalPages:  totalPages,
        },
    }, nil
}

// ─── HELPER ───────────────────────────────────────────────────────────

// validateInput menjalankan validasi dan mengkonversi error ke format proto.
func (uc *ProductUsecase) validateInput(input any) []*commonv1.ValidationError {
    if err := uc.validator.Struct(input); err != nil {
        var validationErrors []*commonv1.ValidationError
        for _, e := range err.(validator.ValidationErrors) {
            validationErrors = append(validationErrors, &commonv1.ValidationError{
                Field:   e.Field(),
                Message: e.Tag() + " validation failed",
            })
        }
        return validationErrors
    }
    return nil
}
```

---

## Phase 8: Handler Layer (gRPC Transport)

### Apa itu Handler Layer?

**Handler Layer** adalah pintu masuk dari dunia luar. Tugasnya sangat spesifik:
1. Menerima request dari gRPC
2. Meneruskan ke Usecase
3. Mengembalikan response

Handler **tidak boleh** berisi business logic apapun.

> **Analogi**: Handler adalah resepsionis — menerima tamu, mengantarkan ke ruangan yang tepat,
> menyampaikan kembali jawaban. Resepsionis tidak membuat keputusan bisnis.

### 8.1 Implementasi Handler

```bash
mkdir -p internal/handler/grpc
```

**File**: `services/product-service/internal/handler/grpc/product_handler.go`

```go
package grpc

import (
    "context"

    productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
    "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/usecase"
)

// Compile-time check: pastikan ProductHandler benar-benar implement interface.
// Jika tidak, error akan muncul saat build, bukan saat runtime.
var _ productv1.ProductServiceServer = (*ProductHandler)(nil)

// ProductHandler mengimplementasikan ProductServiceServer dari hasil generate proto.
type ProductHandler struct {
    // UnimplementedProductServiceServer: jika ada RPC baru di proto tapi belum diimplementasikan,
    // tidak akan panic — akan return "Unimplemented" secara default.
    productv1.UnimplementedProductServiceServer
    uc *usecase.ProductUsecase
}

// New membuat instance ProductHandler baru.
func New(uc *usecase.ProductUsecase) *ProductHandler {
    return &ProductHandler{uc: uc}
}

func (h *ProductHandler) CreateProduct(
    ctx context.Context, req *productv1.CreateProductRequest,
) (*productv1.CreateProductResponse, error) {
    return h.uc.CreateProduct(ctx, req)
}

func (h *ProductHandler) GetProduct(
    ctx context.Context, req *productv1.GetProductRequest,
) (*productv1.GetProductResponse, error) {
    return h.uc.GetProduct(ctx, req)
}

func (h *ProductHandler) UpdateProduct(
    ctx context.Context, req *productv1.UpdateProductRequest,
) (*productv1.UpdateProductResponse, error) {
    return h.uc.UpdateProduct(ctx, req)
}

func (h *ProductHandler) DeleteProduct(
    ctx context.Context, req *productv1.DeleteProductRequest,
) (*productv1.DeleteProductResponse, error) {
    return h.uc.DeleteProduct(ctx, req)
}

func (h *ProductHandler) ListProducts(
    ctx context.Context, req *productv1.ListProductsRequest,
) (*productv1.ListProductsResponse, error) {
    return h.uc.ListProducts(ctx, req)
}
```

---

## Phase 9: Server Wiring

### Apa itu Wiring?

**Wiring** adalah proses merangkai semua komponen:
`Repository → Usecase → Handler → gRPC Server`

Ini dilakukan di `main.go`. File ini harus **sesederhana mungkin** — hanya membuat instance
dan menghubungkannya. Tidak ada business logic di sini!

**File**: `services/product-service/cmd/server/main.go`

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "github.com/jackc/pgx/v5/pgxpool"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/reflection"

    productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
    "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/config"
    grpchandler "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/handler/grpc"
    "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/repository/postgres"
    "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/usecase"
)

func main() {
    // 1. Setup structured logger (JSON format, mudah di-parse oleh log aggregator)
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    slog.SetDefault(logger)

    // 2. Load konfigurasi dari environment variables
    cfg := config.Load()
    slog.Info("config loaded", "env", cfg.AppEnv)

    // 3. Koneksi ke PostgreSQL
    ctx := context.Background()
    dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer dbPool.Close()

    if err := dbPool.Ping(ctx); err != nil {
        slog.Error("database ping failed", "error", err)
        os.Exit(1)
    }
    slog.Info("database connected")

    // 4. Jalankan SQL migrations
    if err := runMigrations(cfg.DatabaseURL); err != nil {
        slog.Error("migration failed", "error", err)
        os.Exit(1)
    }
    slog.Info("migrations applied")

    // 5. Wiring: Repository → Usecase → Handler
    productRepo    := postgres.New(dbPool)            // Implementasi ProductRepository
    productUC      := usecase.New(productRepo)         // Business logic
    productHandler := grpchandler.New(productUC)       // gRPC transport

    // 6. Buat dan konfigurasi gRPC server
    grpcServer := grpc.NewServer(
        // Tempat menambahkan interceptors nanti (logging, auth, recovery)
    )
    productv1.RegisterProductServiceServer(grpcServer, productHandler)
    reflection.Register(grpcServer)  // Aktifkan reflection (untuk Postman/grpcurl)

    // 7. Jalankan gRPC server di goroutine terpisah
    grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
    if err != nil {
        slog.Error("failed to listen", "port", cfg.GRPCPort, "error", err)
        os.Exit(1)
    }
    go func() {
        slog.Info("gRPC server started", "port", cfg.GRPCPort)
        if err := grpcServer.Serve(grpcListener); err != nil {
            slog.Error("gRPC serve error", "error", err)
        }
    }()

    // 8. Setup grpc-gateway (HTTP/REST → gRPC proxy)
    gwMux := runtime.NewServeMux()
    gwOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
    if err := productv1.RegisterProductServiceHandlerFromEndpoint(
        ctx, gwMux,
        fmt.Sprintf("localhost:%s", cfg.GRPCPort),
        gwOpts,
    ); err != nil {
        slog.Error("failed to register gateway", "error", err)
        os.Exit(1)
    }

    httpServer := &http.Server{
        Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
        Handler: gwMux,
    }
    go func() {
        slog.Info("HTTP gateway started", "port", cfg.HTTPPort)
        if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("HTTP server error", "error", err)
        }
    }()

    // 9. Graceful shutdown — tunggu sinyal SIGINT (Ctrl+C) atau SIGTERM
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    slog.Info("shutdown signal received...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = httpServer.Shutdown(shutdownCtx)
    grpcServer.GracefulStop()  // Tunggu semua RPC selesai sebelum mati
    slog.Info("servers stopped gracefully")
}

// runMigrations menjalankan SQL migration yang belum dijalankan.
func runMigrations(databaseURL string) error {
    m, err := migrate.New(
        "file://internal/repository/migrations",
        databaseURL,
    )
    if err != nil {
        return fmt.Errorf("create migrator: %w", err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrate up: %w", err)
    }
    return nil
}
```

### 9.2 Jalankan Service

```bash
# Terminal 1: Jalankan PostgreSQL (jika belum ada)
docker run -d \
  --name store-postgres \
  -e POSTGRES_USER=storeuser \
  -e POSTGRES_PASSWORD=storepass \
  -e POSTGRES_DB=storedb \
  -p 5432:5432 \
  postgres:16-alpine

# Terminal 2: Jalankan service
cd services/product-service
export DATABASE_URL="postgres://storeuser:storepass@localhost:5432/storedb?sslmode=disable"
go run cmd/server/main.go
```

Output yang diharapkan:
```
{"level":"INFO","msg":"config loaded","env":"development"}
{"level":"INFO","msg":"database connected"}
{"level":"INFO","msg":"migrations applied"}
{"level":"INFO","msg":"gRPC server started","port":"50051"}
{"level":"INFO","msg":"HTTP gateway started","port":"8080"}
```

---

## Phase 10: Testing

### 10.1 Via REST (curl / Postman HTTP)

```bash
# Create Product
curl -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop Gaming","description":"Gaming laptop","price_cents":15000000,"stock":10}'

# List Products (dengan pagination)
curl "http://localhost:8080/v1/products?pagination.page=1&pagination.page_size=5"

# Get Product (ganti UUID_DI_SINI)
curl http://localhost:8080/v1/products/UUID_DI_SINI

# Partial Update — hanya update name dan stock
curl -X PATCH http://localhost:8080/v1/products/UUID_DI_SINI \
  -H "Content-Type: application/json" \
  -d '{"update_mask":"name,stock","product":{"name":"Laptop Pro","stock":5}}'

# Delete
curl -X DELETE http://localhost:8080/v1/products/UUID_DI_SINI
```

### 10.2 Via gRPC (grpcurl)

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List semua service (butuh reflection aktif)
grpcurl -plaintext localhost:50051 list

# Create Product via gRPC
grpcurl -plaintext -d '{
  "name": "Headphone",
  "description": "Noise-cancelling",
  "price_cents": 2990000,
  "stock": 50
}' localhost:50051 product.v1.ProductService/CreateProduct

# List dengan pagination
grpcurl -plaintext -d '{"pagination":{"page":1,"page_size":10}}' \
  localhost:50051 product.v1.ProductService/ListProducts
```

### 10.3 Via Swagger UI (Browser)

```bash
# Buka https://editor.swagger.io/
# Klik File → Import File
# Pilih: gen/openapiv2/api.swagger.json
# Ubah server URL ke: http://localhost:8080
# Test langsung dari browser!
```

### 10.4 Via Postman (GUI)

```
1. Buka Postman → New → gRPC Request
2. URL: localhost:50051
3. Klik "Import a .proto file"
   → Pilih: proto/product/v1/product.proto
   → Set import path ke folder: /path/to/simple-store-go-grpc/proto/
4. Pilih method dari dropdown
5. Isi payload JSON, klik Invoke
```

---

## Phase 11: Docker & docker-compose

### 11.1 Dockerfile

**File**: `services/product-service/Dockerfile`

```dockerfile
# ── Stage 1: Build ───────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
WORKDIR /app
# Copy go.mod dulu (cache Docker layer untuk dependencies)
COPY go.mod go.sum ./
RUN go mod download
# Copy source code
COPY . .
# Build binary (CGO_ENABLED=0 = binary tidak butuh C library)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/server/...

# ── Stage 2: Runtime (image kecil) ──────────────────────────────
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/internal/repository/migrations ./internal/repository/migrations
EXPOSE 50051 8080
CMD ["./server"]
```

### 11.2 docker-compose

**File**: `docker-compose.yml` (di root monorepo)

```yaml
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    container_name: store-postgres
    environment:
      POSTGRES_USER: storeuser
      POSTGRES_PASSWORD: storepass
      POSTGRES_DB: storedb
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U storeuser -d storedb"]
      interval: 5s
      timeout: 5s
      retries: 5

  product-service:
    build:
      context: ./services/product-service
      dockerfile: Dockerfile
    container_name: product-service
    environment:
      GRPC_PORT: "50051"
      HTTP_PORT: "8080"
      DATABASE_URL: "postgres://storeuser:storepass@postgres:5432/storedb?sslmode=disable"
      APP_ENV: "development"
      LOG_LEVEL: "debug"
    ports:
      - "50051:50051"
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres-data:
```

```bash
# Jalankan semua dengan docker-compose
docker compose up -d

# Lihat log
docker compose logs -f product-service

# Stop
docker compose down

# Stop + hapus data
docker compose down -v
```

---

## Rangkuman: Alur Data End-to-End

```
Client (curl / Postman / grpcurl)
    │
    │  POST http://localhost:8080/v1/products
    ▼
grpc-gateway (HTTP → gRPC converter)
    │
    │  gRPC: CreateProduct(CreateProductRequest)
    ▼
ProductHandler  [handler/grpc/]
    │  Terima request, teruskan ke usecase
    ▼
ProductUsecase  [usecase/]
    │  1. Validasi input
    │  2. Buat domain.Product{ID: uuid, Name: ..., PriceCents: ...}
    │  3. Panggil repo.Create()
    │  4. Build CreateProductResponse
    ▼
ProductRepo     [repository/postgres/]
    │  INSERT INTO products VALUES ($1, $2, ...)
    ▼
PostgreSQL
    │  Simpan data, return created_at, updated_at
    ▲
    │
ProductRepo → ProductUsecase → ProductHandler → grpc-gateway → Client

    Response:
    {
      "base": { "http_status_code": 201, "is_success": true, "message": "..." },
      "product": { "id": "uuid", "name": "Laptop", "price_cents": "15000000", ... }
    }
```

---

## Tabel Ringkasan Layer

| Layer | Folder | Tugasnya | Bergantung pada |
|-------|--------|----------|-----------------|
| **Handler** | `internal/handler/grpc/` | Terima request gRPC, teruskan ke UseCase | UseCase |
| **UseCase** | `internal/usecase/` | Business logic, validasi, orchestration | Domain Interface |
| **Domain** | `internal/domain/` | Entitas bisnis + repository interface | Standard library saja |
| **Repository** | `internal/repository/postgres/` | Implementasi database | Domain Interface |
| **Config** | `internal/config/` | Baca konfigurasi dari env | Standard library saja |
| **Main** | `cmd/server/main.go` | Wiring semua layer + start server | Semua layer |

---

## Best Practices yang Diterapkan

| Praktik | Penjelasan |
|---------|------------|
| **Schema-First** | Proto adalah sumber kebenaran API, bukan kode Go |
| **Clean Architecture** | Dependency hanya mengalir ke dalam: Handler→UC→Domain←Repo |
| **price_cents (int64)** | Integer untuk uang — floating point tidak akurat |
| **google.protobuf.Timestamp** | Type-safe, bukan string |
| **FieldMask untuk Update** | Partial update yang aman dan benar |
| **Domain Errors** | Error bisnis terpisah dari error infrastruktur |
| **Parameterized Query** | `$1, $2, ...` — mencegah SQL Injection |
| **Soft Delete** | `deleted_at` kolom — data tidak benar-benar hilang |
| **Connection Pooling** | `pgxpool` untuk efisiensi koneksi database |
| **Graceful Shutdown** | Server menyelesaikan request yang sedang berjalan |
| **Structured Logging** | `log/slog` JSON format, mudah di-parse |
| **Environment Config** | Semua dari env vars, tidak ada yang hardcode |
| **buf v2 + managed mode** | `go_package` tidak perlu ditulis manual di .proto |
| **Multi-stage Docker** | Image production kecil dan aman |
