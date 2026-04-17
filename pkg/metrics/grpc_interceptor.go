package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// GRPCInterceptor returns a gRPC unary server interceptor that records metrics
func (m *Metrics) GRPCInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		method := info.FullMethod

		// Track in-flight requests
		m.grpcRequestsInFlight.WithLabelValues(method).Inc()
		defer m.grpcRequestsInFlight.WithLabelValues(method).Dec()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start).Seconds()
		code := status.Code(err).String()

		m.grpcRequestsTotal.WithLabelValues(method, code).Inc()
		m.grpcRequestDuration.WithLabelValues(method).Observe(duration)

		return resp, err
	}
}
