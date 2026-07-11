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

// PERBAIKAN: Menggunakan domain.ProductListParams
func (r *productRepo) List(ctx context.Context, params domain.ProductListParams) ([]*domain.Product, int64, error) {
	// 1. Inisialisasi dasar kueri
	query := `
		SELECT
			id, name, description, price_cents, stock, created_at, updated_at, created_by, updated_by,
			COUNT(*) OVER() AS total_items
		FROM products
		WHERE deleted_at IS NULL
	`

	// 2. Slice untuk menyimpan argumen dinamis ($1, $2, dst)
	var args []interface{}
	argCount := 1

	// 3. Pembangunan kueri dinamis (Dynamic Query Building)
	if params.SearchName != "" {
		// Menggunakan ILIKE untuk pencarian nama case-insensitive
		query += fmt.Sprintf(" AND name ILIKE $%d", argCount)
		args = append(args, "%"+params.SearchName+"%")
		argCount++
	}

	if params.MinPrice != nil {
		query += fmt.Sprintf(" AND price_cents >= $%d", argCount)
		args = append(args, *params.MinPrice)
		argCount++
	}

	// 4. Tambahkan Order, Limit, dan Offset
	offset := (params.Page - 1) * params.PageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, params.PageSize, offset)

	// 5. Eksekusi Kueri
	rows, err := r.db.Query(ctx, query, args...)
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
			&total,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("repo.List scan: %w", err)
		}
		products = append(products, p)
	}

	return products, total, nil
}
