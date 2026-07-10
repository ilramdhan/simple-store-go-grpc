package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	userIDKey    ctxKey = "user_id"
)

// InitLogger mengatur default logger global.
func InitLogger(levelStr, serviceName string) {
	var level slog.Level
	// Jika level salah atau kosong, otomatis fallback ke Info
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		// Wajib untuk production: Menambahkan "source" (file & baris kode) ke dalam log
		AddSource: true,
	}

	// Gunakan JSON handler dengan default atribut service name
	handler := slog.NewJSONHandler(os.Stdout, opts).WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
	})

	slog.SetDefault(slog.New(handler))
}

// ContextWithRequestID adalah helper bagi Middleware untuk menyuntikkan Request ID
func ContextWithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKey, reqID)
}

// ContextWithUserID adalah helper bagi Middleware Auth untuk menyuntikkan User ID
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// FromContext mengambil logger dengan atribut dinamis (request_id, user_id) jika ada.
func FromContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()

	// Ekstrak request_id jika tersedia
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		logger = logger.With(slog.String("request_id", reqID))
	}

	// Ekstrak user_id jika tersedia (Sangat berguna untuk melacak error spesifik user)
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		logger = logger.With(slog.String("user_id", userID))
	}

	return logger
}
