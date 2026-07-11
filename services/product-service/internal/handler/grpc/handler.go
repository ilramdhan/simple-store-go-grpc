package grpc

import (
	"context"

	commonv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/common/v1"
	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

// 1. Compile-Time Interface Check (Best Practice)
var _ productv1.ProductServiceServer = (*Handler)(nil)

type Handler struct {
	// 2. Forward Compatibility Placeholder
	productv1.UnimplementedProductServiceServer

	// 3. Dependency ke layer bisnis (Usecase/Service)
	svc domain.ProductService
}

func NewHandler(svc domain.ProductService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	input := domain.CreateProductInput{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		PriceCents:  req.GetPriceCents(),
		Stock:       req.GetStock(),
	}

	product, err := h.svc.Create(ctx, input)
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
	product, err := h.svc.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &productv1.GetProductResponse{
		Base:    &commonv1.BaseResponse{HttpStatusCode: 200, IsSuccess: true, Message: "Success"},
		Product: domainToProtoProduct(product),
	}, nil
}

func (h *Handler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error) {
	// 4. Mapping Partial Update (Pointer fields)
	// Asumsi: di file .proto Anda menggunakan kata kunci 'optional' untuk field update ini.
	input := domain.UpdateProductInput{
		Name:        req.Name, // Berupa pointer *string jika optional
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Stock:       req.Stock,
	}

	product, err := h.svc.Update(ctx, req.GetId(), input)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &productv1.UpdateProductResponse{
		Base:    &commonv1.BaseResponse{HttpStatusCode: 200, IsSuccess: true, Message: "Product updated successfully"},
		Product: domainToProtoProduct(product),
	}, nil
}

func (h *Handler) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductResponse, error) {
	if err := h.svc.Delete(ctx, req.GetId()); err != nil {
		return nil, domainErrorToGRPC(err)
	}

	return &productv1.DeleteProductResponse{
		Base: &commonv1.BaseResponse{HttpStatusCode: 200, IsSuccess: true, Message: "Product deleted successfully"},
	}, nil
}

func (h *Handler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	// 5. Mapping Pagination & Filtering
	params := domain.ProductListParams{
		Page:       req.GetPage(),
		PageSize:   req.GetPageSize(),
		SearchName: req.GetSearchName(),
		MinPrice:   req.MinPrice, // Pointer *int64
	}

	products, total, err := h.svc.List(ctx, params)
	if err != nil {
		return nil, domainErrorToGRPC(err)
	}

	// 6. Mapping Slice of Objects
	var protoProducts []*productv1.Product
	for _, p := range products {
		protoProducts = append(protoProducts, domainToProtoProduct(p))
	}

	// Menghitung jumlah halaman (TotalPages)
	totalPages := int32(0)
	if params.PageSize > 0 {
		totalPages = int32((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	}

	return &productv1.ListProductsResponse{
		Base:       &commonv1.BaseResponse{HttpStatusCode: 200, IsSuccess: true, Message: "Success"},
		Products:   protoProducts,
		Pagination: &commonv1.PaginationResponse{
			CurrentPage: params.Page,
			PageSize:    params.PageSize,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}
