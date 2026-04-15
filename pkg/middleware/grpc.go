package middleware

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/monorepo/pkg/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// Header keys for trace propagation
	requestIDHeader = "x-request-id"
	traceIDHeader   = "x-trace-id"
)

// LoggerInterceptor returns a gRPC unary interceptor that logs requests
func LoggerInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Generate or extract request ID
		requestID := getOrCreateID(ctx, requestIDHeader)
		ctx = logging.SetRequestID(ctx, requestID)

		// Generate or extract trace ID
		traceID := getOrCreateTraceID(ctx)
		ctx = logging.SetTraceID(ctx, traceID)

		// Create context-aware logger with request-scoped fields
		reqLogger := logger.WithContext(ctx)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", code.String()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			reqLogger.Error("grpc request", fields...)
		} else {
			reqLogger.Info("grpc request", fields...)
		}

		return resp, err
	}
}

// getOrCreateID extracts ID from metadata or generates a new one
func getOrCreateID(ctx context.Context, headerKey string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get(headerKey); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return uuid.New().String()
}

// getOrCreateTraceID extracts trace ID from metadata or generates a new one
func getOrCreateTraceID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get(traceIDHeader); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return uuid.New().String()
}

// RecoveryInterceptor returns a gRPC unary interceptor that recovers from panics
func RecoveryInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				reqLogger := logger.WithContext(ctx)
				reqLogger.Error("panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// ChainUnaryInterceptors chains multiple unary interceptors into one
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return current(currentCtx, currentReq, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(ctx, req)
	}
}

// PropagationInterceptor returns a gRPC client interceptor that propagates
// request_id and trace_id from context to outgoing metadata
func PropagationInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Extract trace info from context
		requestID := logging.GetRequestID(ctx)
		traceID := logging.GetTraceID(ctx)

		// Only propagate if we have values
		if requestID != "" || traceID != "" {
			// Get existing outgoing metadata or create new
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				md = metadata.New(nil)
			} else {
				// Copy to avoid modifying the original
				md = md.Copy()
			}

			if requestID != "" {
				md.Set(requestIDHeader, requestID)
			}
			if traceID != "" {
				md.Set(traceIDHeader, traceID)
			}

			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
