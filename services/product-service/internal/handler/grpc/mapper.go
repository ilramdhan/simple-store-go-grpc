package grpc

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/common/v1"
	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/domain"
)

// domainToProtoProduct memetakan entitas internal ke struktur pesan Protobuf.
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

// domainErrorToGRPC menerjemahkan error domain bisnis menjadi status gRPC berstandar Google (Rich Errors).
func domainErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	// 1. Tangani Error Validasi (Menggunakan Custom Type dari Domain)
	// Kita gunakan Type Assertion karena tipe error-nya sudah jelas berupa slice struct kita sendiri.
	if vErrs, ok := err.(domain.ValidationErrors); ok {
		// Inisiasi status dasar
		st := status.New(codes.InvalidArgument, "invalid request parameters")

		// Siapkan wadah untuk detail pelanggaran berstandar Google
		var violations []*errdetails.BadRequest_FieldViolation
		for _, v := range vErrs {
			violations = append(violations, &errdetails.BadRequest_FieldViolation{
				Field:       v.Field,
				Description: v.Message,
			})
		}

		// Tempelkan detail pelanggaran ke dalam status gRPC
		br := &errdetails.BadRequest{FieldViolations: violations}
		stWithDetails, attachErr := st.WithDetails(br)
		if attachErr != nil {
			// Fallback aman jika gagal menempelkan detail
			return st.Err()
		}

		return stWithDetails.Err()
	}

	// 2. Tangani error Sentinel (State Bisnis)
	if errors.Is(err, domain.ErrProductNotFound) {
		return status.Errorf(codes.NotFound, "product with specified ID not found")
	}

	// Jika ada domain error lain seperti ErrConflict, tambahkan di sini:
	// if errors.Is(err, domain.ErrConflict) { return status.Errorf(codes.AlreadyExists, "data already exists") }

	// 3. Fallback (Penyembunyian Jejak)
	// Jika error tidak dikenali (seperti database mati), berikan Internal Server Error
	// tanpa membocorkan pesan error asli ke sisi client.
	return status.Errorf(codes.Internal, "an unexpected internal server error occurred")
}
