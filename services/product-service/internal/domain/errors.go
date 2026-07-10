package domain

import (
	"errors"
	"fmt"
)

// 1. Sentinel Errors (Hanya untuk State Bisnis / Database)
var (
	ErrProductNotFound = errors.New("product not found")
	ErrInternalServer  = errors.New("internal server error")
	// Tidak perlu lagi ErrProductNameEmpty, ErrProductPriceNegative, dll.
)

// 2. Custom Error Struct (Untuk Validasi Dinamis)
type ValidationError struct {
	Field   string
	Message string
}

// 3. Kumpulan ValidationError agar bisa mengembalikan banyak error sekaligus
type ValidationErrors []ValidationError

// Mengimplementasikan interface `error` bawaan Golang
func (ve ValidationErrors) Error() string {
	if len(ve) > 0 {
		return fmt.Sprintf("validation failed on field: %s, condition: %s", ve[0].Field, ve[0].Message)
	}
	return "validation failed"
}
