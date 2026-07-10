package domain

import (
	"context"
	"time"
)

type Product struct {
	ID          string
	Name        string
	Description string
	PriceCents  int64
	Stock       int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // Menggunakan pointer, nil berarti belum dihapus (Soft Delete)
	CreatedBy   string
	UpdatedBy   string
}

// Struct khusus untuk pencarian dan pagination yang dinamis
type ProductListParams struct {
	Page       int32
	PageSize   int32
	SearchName string // Opsional: Untuk filter nama
	MinPrice   *int64 // Opsional: Untuk filter harga
}

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
	// Gunakan struct sebagai parameter
	List(ctx context.Context, params ProductListParams) ([]*Product, int64, error)
}

type ProductService interface {
	Create(ctx context.Context, input CreateProductInput) (*Product, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, input UpdateProductInput) (*Product, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, params ProductListParams) ([]*Product, int64, error)
}

type CreateProductInput struct {
	Name        string `validate:"required,min=3,max=100"` // Spasi dihapus
	Description string `validate:"max=1000"`
	PriceCents  int64  `validate:"required,min=1"`
	Stock       int32  `validate:"required,min=0"`
}

type UpdateProductInput struct {
	Name        *string `validate:"omitempty,min=3,max=100"`
	Description *string `validate:"omitempty,max=500"`
	PriceCents  *int64  `validate:"omitempty,min=0"`
	Stock       *int32  `validate:"omitempty,min=0"`
}
