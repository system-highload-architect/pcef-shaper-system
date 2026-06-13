package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type LoggerInterface interface {
	Info(format string, v ...any)
	Error(format string, v ...any)
}

// UnaryServerInterceptor — сквозной замер Latency и аудит ошибок L7 уровня
func UnaryServerInterceptor(log LoggerInterface) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		startTime := time.Now()

		// Передаем управление дальше по call-stack в бизнес-логику
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		st, _ := status.FromError(err)

		if err != nil {
			log.Error("RPC FAILED [Method: %s] | Latency: %v | Code: %s | Message: %s",
				info.FullMethod, duration, st.Code().String(), st.Message())
		} else {
			log.Info("RPC SUCCESS [Method: %s] | Latency: %v | Code: %s",
				info.FullMethod, duration, st.Code().String())
		}

		return resp, err
	}
}
