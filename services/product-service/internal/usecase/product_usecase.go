package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
)

type productUsecase struct {
	repo     domain.ProductRepository
	validate *validator.Validate
}

func NewProductUsecase(repo domain.ProductRepository) domain.ProductService {
	return &productUsecase{
		repo:     repo,
		validate: validator.New(),
	}
}

// helper untuk menerjemahkan error validator bawaan menjadi domain error kita
func translateValidationErrors(err error) error {
	if valErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
		var out domain.ValidationErrors
		for _, fe := range valErrors {
			out = append(out, domain.ValidationError{
				Field:   fe.Field(),
				Message: fe.Tag(),
			})
		}
		return out
	}
	return err
}

// mock helper: Di aplikasi asli, kita mengekstrak ini dari context JWT/Auth
func getUserFromContext(ctx context.Context) string {
	// TODO: Nanti diisi dengan logika auth interceptor
	return "system-admin"
}

func (u *productUsecase) Create(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error) {
	// 1. Validasi Input
	if err := u.validate.Struct(input); err != nil {
		return nil, translateValidationErrors(err)
	}

	userID := getUserFromContext(ctx)

	// 2. Pembentukan Entitas (Business Logic)
	p := &domain.Product{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		PriceCents:  input.PriceCents,
		Stock:       input.Stock,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	// 3. Simpan melalui Repository
	if err := u.repo.Create(ctx, p); err != nil {
		logger.FromContext(ctx).Error("failed to create product", slog.String("error", err.Error()))
		return nil, domain.ErrInternalServer
	}

	return p, nil
}

func (u *productUsecase) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *productUsecase) Update(ctx context.Context, id string, input domain.UpdateProductInput) (*domain.Product, error) {
	// 1. Validasi Input Parsial
	if err := u.validate.Struct(input); err != nil {
		return nil, translateValidationErrors(err)
	}

	// 2. Cek eksistensi data sebelum update
	p, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err // Akan me-return ErrProductNotFound dari repo
	}

	// 3. Terapkan Partial Update (Nil check)
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
	p.UpdatedBy = getUserFromContext(ctx)

	// 4. Simpan ke database
	if err := u.repo.Update(ctx, p); err != nil {
		logger.FromContext(ctx).Error("failed to update product", slog.String("error", err.Error()))
		return nil, domain.ErrInternalServer
	}

	return p, nil
}

func (u *productUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}

func (u *productUsecase) List(ctx context.Context, params domain.ProductListParams) ([]*domain.Product, int64, error) {
	// Fallback/Default values untuk Pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 10 // Standar limit agar DB tidak ditarik terlalu berat
	}

	return u.repo.List(ctx, params)
}
