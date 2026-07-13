package interceptor

import (
	"context"
	"time"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
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
