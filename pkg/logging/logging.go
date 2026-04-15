package logging

import (
	"context"
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context keys for request-scoped values
type ctxKey string

const (
	RequestIDKey ctxKey = "request_id"
	TraceIDKey   ctxKey = "trace_id"
)

// Logger wraps zap.Logger for convenience
type Logger struct {
	*zap.Logger
	service string
}

// New creates a new logger based on the provided configuration
func New(level, format, service string) *Logger {
	var config zap.Config

	if format == "console" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Enable caller info (file and line number)
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	// Add service name as a default field
	if service != "" {
		logger = logger.With(zap.String("service", service))
	}

	return &Logger{Logger: logger, service: service}
}

// Named returns a logger with the provided name
func (l *Logger) Named(name string) *Logger {
	return &Logger{Logger: l.Logger.Named(name), service: l.service}
}

// With returns a logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...), service: l.service}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// WithContext returns a logger with request-scoped fields from context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make([]zap.Field, 0, 2)

	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields = append(fields, zap.String("request_id", requestID.(string)))
	}
	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		fields = append(fields, zap.String("trace_id", traceID.(string)))
	}

	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}

// SetRequestID sets the request ID in the context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// SetTraceID sets the trace ID in the context
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(TraceIDKey); v != nil {
		return v.(string)
	}
	return ""
}
